package sync

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/o-mid/contract-ops/api/internal/connections"
	"github.com/o-mid/contract-ops/api/internal/platform/auth"
	"github.com/o-mid/contract-ops/api/internal/platform/problem"
)

type Handler struct {
	Jobs        *Store
	Connections *connections.Store
}

func NewHandler(jobs *Store, conns *connections.Store) Handler {
	return Handler{Jobs: jobs, Connections: conns}
}

func (h Handler) Routes(router chi.Router) {
	router.Post("/v1/connections/{id}/backfill", h.backfill)
	router.Post("/v1/sync/jobs/{id}/cancel", h.cancel)
}

type backfillBody struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func (h Handler) backfill(writer http.ResponseWriter, request *http.Request) {
	workspaceID := auth.WorkspaceID(request.Context())
	if workspaceID == "" {
		problem.Write(writer, http.StatusUnauthorized, "auth_invalid", "missing bearer token")
		return
	}
	var body backfillBody
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		problem.Write(writer, http.StatusBadRequest, "", "invalid json")
		return
	}
	chunks := Chunks(body.Start, body.End)
	if len(chunks) == 0 || len(chunks) > 366 {
		problem.Write(writer, http.StatusBadRequest, "", "backfill must cover 1 to 366 days")
		return
	}
	conn, err := h.Connections.Get(request.Context(), workspaceID, chi.URLParam(request, "id"))
	if err != nil {
		problem.Write(writer, http.StatusNotFound, "", "connection not found")
		return
	}
	jobs := make([]Job, 0, len(chunks))
	for _, chunk := range chunks {
		job, err := h.Jobs.Enqueue(request.Context(), Job{
			ConnectionID: conn.ID,
			WorkspaceID:  workspaceID,
			Kind:         "backfill",
			WindowStart:  chunk[0],
			WindowEnd:    chunk[1],
		})
		if err != nil {
			problem.Write(writer, http.StatusInternalServerError, "", "could not enqueue backfill")
			return
		}
		jobs = append(jobs, job)
	}
	writeJSON(writer, http.StatusAccepted, struct {
		Jobs []Job `json:"jobs"`
	}{Jobs: jobs})
}

func (h Handler) cancel(writer http.ResponseWriter, request *http.Request) {
	workspaceID := auth.WorkspaceID(request.Context())
	if workspaceID == "" {
		problem.Write(writer, http.StatusUnauthorized, "auth_invalid", "missing bearer token")
		return
	}
	job, err := h.Jobs.Cancel(request.Context(), workspaceID, chi.URLParam(request, "id"))
	if err != nil {
		problem.Write(writer, http.StatusNotFound, "", "job not found")
		return
	}
	writeJSON(writer, http.StatusOK, job)
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
