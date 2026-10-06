package connections

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/o-mid/contract-ops/api/internal/platform/auth"
	"github.com/o-mid/contract-ops/api/internal/platform/problem"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) Handler {
	return Handler{svc: svc}
}

func (h Handler) Routes(router chi.Router) {
	router.Post("/v1/connections", h.create)
	router.Get("/v1/connections", h.list)
	router.Get("/v1/connections/{id}", h.get)
	router.Patch("/v1/connections/{id}", h.patch)
	router.Post("/v1/connections/{id}/verify", h.verify)
	router.Post("/v1/connections/{id}/rotate", h.rotate)
}

type patchBody struct {
	Name   *string `json:"name"`
	Paused *bool   `json:"paused"`
}

func (h Handler) create(writer http.ResponseWriter, request *http.Request) {
	workspaceID, ok := workspace(writer, request)
	if !ok {
		return
	}
	var body CreateInput
	if !decode(writer, request, &body) {
		return
	}
	conn, err := h.svc.Create(request.Context(), workspaceID, body)
	if err != nil {
		writeErr(writer, err)
		return
	}
	writeJSON(writer, http.StatusCreated, conn)
}

func (h Handler) list(writer http.ResponseWriter, request *http.Request) {
	workspaceID, ok := workspace(writer, request)
	if !ok {
		return
	}
	conns, err := h.svc.List(request.Context(), workspaceID)
	if err != nil {
		writeErr(writer, err)
		return
	}
	if conns == nil {
		conns = []Connection{}
	}
	writeJSON(writer, http.StatusOK, struct {
		Connections []Connection `json:"connections"`
	}{Connections: conns})
}

func (h Handler) get(writer http.ResponseWriter, request *http.Request) {
	workspaceID, ok := workspace(writer, request)
	if !ok {
		return
	}
	conn, err := h.svc.Get(request.Context(), workspaceID, chi.URLParam(request, "id"))
	if err != nil {
		writeErr(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, conn)
}

func (h Handler) patch(writer http.ResponseWriter, request *http.Request) {
	workspaceID, ok := workspace(writer, request)
	if !ok {
		return
	}
	var body patchBody
	if !decode(writer, request, &body) {
		return
	}
	if body.Name == nil && body.Paused == nil {
		problem.Write(writer, http.StatusBadRequest, "", "name or paused is required")
		return
	}
	id := chi.URLParam(request, "id")
	conn, err := h.svc.Get(request.Context(), workspaceID, id)
	if err != nil {
		writeErr(writer, err)
		return
	}
	if body.Name != nil {
		conn, err = h.svc.Rename(request.Context(), workspaceID, id, *body.Name)
		if err != nil {
			writeErr(writer, err)
			return
		}
	}
	if body.Paused != nil {
		conn, err = h.svc.SetPaused(request.Context(), workspaceID, id, *body.Paused)
		if err != nil {
			writeErr(writer, err)
			return
		}
	}
	writeJSON(writer, http.StatusOK, conn)
}

func (h Handler) verify(writer http.ResponseWriter, request *http.Request) {
	workspaceID, ok := workspace(writer, request)
	if !ok {
		return
	}
	conn, err := h.svc.Verify(request.Context(), workspaceID, chi.URLParam(request, "id"))
	if err != nil {
		writeErr(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, conn)
}

func (h Handler) rotate(writer http.ResponseWriter, request *http.Request) {
	workspaceID, ok := workspace(writer, request)
	if !ok {
		return
	}
	var body RotateInput
	if !decode(writer, request, &body) {
		return
	}
	conn, err := h.svc.Rotate(request.Context(), workspaceID, chi.URLParam(request, "id"), body)
	if err != nil {
		writeErr(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, conn)
}

func workspace(writer http.ResponseWriter, request *http.Request) (string, bool) {
	id := auth.WorkspaceID(request.Context())
	if id == "" {
		problem.Write(writer, http.StatusUnauthorized, "auth_invalid", "missing bearer token")
		return "", false
	}
	return id, true
}

func decode(writer http.ResponseWriter, request *http.Request, dest any) bool {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		problem.Write(writer, http.StatusBadRequest, "", "invalid json")
		return false
	}
	return true
}

func writeErr(writer http.ResponseWriter, err error) {
	var failure *Failure
	if errors.As(err, &failure) {
		problem.Write(writer, failure.Status, failure.Code, failure.Detail)
		return
	}
	if errors.Is(err, ErrNotFound) {
		problem.Write(writer, http.StatusNotFound, "", "connection not found")
		return
	}
	problem.Write(writer, http.StatusInternalServerError, "", "internal error")
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
