package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/o-mid/contract-ops/api/internal/events"
)

type Server struct {
	store *events.Store
}

func NewServer(store *events.Store) Server {
	return Server{store: store}
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /v1/events", s.listEvents)
	mux.HandleFunc("GET /v1/events/stream", s.streamEvents)

	return withCORS(mux)
}

func (s Server) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) listEvents(writer http.ResponseWriter, request *http.Request) {
	status := request.URL.Query().Get("status")
	if !validStatus(status) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{
			"error": "status must be processed, pending, failed, or all",
		})
		return
	}

	page := s.store.List(request.URL.Query().Get("q"), status)
	writeJSON(writer, http.StatusOK, page)
}

func (s Server) streamEvents(writer http.ResponseWriter, request *http.Request) {
	status := request.URL.Query().Get("status")
	if !validStatus(status) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{
			"error": "status must be processed, pending, failed, or all",
		})
		return
	}

	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.WriteHeader(http.StatusOK)
	flusher.Flush()

	query := request.URL.Query().Get("q")
	changes, cancel := s.store.Subscribe()
	defer cancel()

	writePage := func() bool {
		page := s.store.List(query, status)
		payload, err := json.Marshal(page)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(writer, "data: %s\n\n", payload); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !writePage() {
		return
	}

	for {
		select {
		case <-request.Context().Done():
			return
		case _, open := <-changes:
			if !open {
				return
			}
			if !writePage() {
				return
			}
		}
	}
}

func validStatus(status string) bool {
	switch status {
	case "", "all", string(events.StatusProcessed), string(events.StatusPending), string(events.StatusFailed):
		return true
	default:
		return false
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		writer.Header().Set("Vary", "Origin")
		next.ServeHTTP(writer, request)
	})
}

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
