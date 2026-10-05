package httpapi

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/o-mid/contract-ops/api/internal/events"
	"github.com/o-mid/contract-ops/api/internal/platform/httpx"
)

const (
	defaultRequestTimeout = 30 * time.Second
	defaultBodyLimitBytes = 1 << 20
)

// Options configures middleware around the existing event routes.
type Options struct {
	Logger         *slog.Logger
	CORSOrigin     string
	RequestTimeout time.Duration
	BodyLimitBytes int64
}

func (o Options) withDefaults() Options {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.CORSOrigin == "" {
		o.CORSOrigin = "http://localhost:5173"
	}
	if o.RequestTimeout <= 0 {
		o.RequestTimeout = defaultRequestTimeout
	}
	if o.BodyLimitBytes <= 0 {
		o.BodyLimitBytes = defaultBodyLimitBytes
	}
	return o
}

type Server struct {
	store *events.Store
	opts  Options
}

func NewServer(store *events.Store, opts Options) Server {
	return Server{store: store, opts: opts.withDefaults()}
}

func (s Server) Handler() http.Handler {
	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.CORS(s.opts.CORSOrigin))
	router.Use(httpx.AccessLog(s.opts.Logger))
	router.Use(httpx.Recover(s.opts.Logger))
	router.Use(httpx.BodyLimit(s.opts.BodyLimitBytes))
	router.Use(httpx.Timeout(s.opts.RequestTimeout, func(request *http.Request) bool {
		// A request timeout would close the stream and the UI would treat
		// that as a failed connection. The client closes it instead.
		return request.URL.Path == "/v1/events/stream"
	}))
	router.Get("/healthz", s.health)
	router.Get("/v1/events", s.listEvents)
	router.Get("/v1/events/stream", s.streamEvents)
	return router
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

func writeJSON(writer http.ResponseWriter, status int, body any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
