package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/o-mid/contract-ops/api/internal/events"
	"github.com/o-mid/contract-ops/api/internal/platform/httpx"
	"github.com/o-mid/contract-ops/api/internal/platform/problem"
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
	// Ready reports whether the process should receive traffic. Nil means not ready.
	Ready func(ctx context.Context) error
	// Metrics serves Prometheus text. Nil leaves /metrics unregistered.
	Metrics http.Handler
	// Instrument records RED metrics. It must pass Flush through for the event stream.
	Instrument func(http.Handler) http.Handler
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
	store events.Feed
	opts  Options
}

func NewServer(store events.Feed, opts Options) Server {
	return Server{store: store, opts: opts.withDefaults()}
}

func (s Server) Handler() http.Handler {
	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.CORS(s.opts.CORSOrigin))
	router.Use(httpx.AccessLog(s.opts.Logger))
	router.Use(httpx.Recover(s.opts.Logger))
	if s.opts.Instrument != nil {
		router.Use(s.opts.Instrument)
	}
	router.Use(httpx.BodyLimit(s.opts.BodyLimitBytes))
	router.Use(httpx.Timeout(s.opts.RequestTimeout, func(request *http.Request) bool {
		// A request timeout would close the stream and the UI would treat
		// that as a failed connection. The client closes it instead.
		return request.URL.Path == "/v1/events/stream"
	}))
	router.Get("/healthz", s.health)
	router.Get("/readyz", s.ready)
	if s.opts.Metrics != nil {
		router.Handle("/metrics", s.opts.Metrics)
	}
	router.Get("/v1/events", s.listEvents)
	router.Get("/v1/events/stream", s.streamEvents)
	return router
}

func (s Server) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) ready(writer http.ResponseWriter, request *http.Request) {
	if s.opts.Ready == nil {
		problem.Write(writer, http.StatusServiceUnavailable, "", "readiness check is not configured")
		return
	}
	if err := s.opts.Ready(request.Context()); err != nil {
		problem.Write(writer, http.StatusServiceUnavailable, "", err.Error())
		return
	}
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) listEvents(writer http.ResponseWriter, request *http.Request) {
	query, ok := eventQuery(writer, request)
	if !ok {
		return
	}

	page, err := s.store.List(request.Context(), query)
	if err != nil {
		writeStoreError(writer, err)
		return
	}
	writeJSON(writer, http.StatusOK, page)
}

func (s Server) streamEvents(writer http.ResponseWriter, request *http.Request) {
	query, ok := eventQuery(writer, request)
	if !ok {
		return
	}

	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	changes, cancel, err := s.store.Subscribe(request.Context())
	if err != nil {
		writeStoreError(writer, err)
		return
	}
	defer cancel()

	writePage := func() bool {
		page, err := s.store.List(request.Context(), query)
		if err != nil {
			return false
		}
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

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.WriteHeader(http.StatusOK)
	flusher.Flush()

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

func eventQuery(writer http.ResponseWriter, request *http.Request) (events.Query, bool) {
	status := request.URL.Query().Get("status")
	if !validStatus(status) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{
			"error": "status must be processed, pending, failed, or all",
		})
		return events.Query{}, false
	}

	limit, err := parseLimit(request.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return events.Query{}, false
	}

	connectionID := request.URL.Query().Get("connection_id")
	if len(connectionID) > 128 {
		writeJSON(writer, http.StatusBadRequest, map[string]string{
			"error": "connection_id is too long",
		})
		return events.Query{}, false
	}

	return events.Query{
		Text:         request.URL.Query().Get("q"),
		Status:       status,
		ConnectionID: connectionID,
		Cursor:       request.URL.Query().Get("cursor"),
		Limit:        limit,
	}, true
}

func parseLimit(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > events.MaxLimit {
		return 0, errors.New("limit must be from 1 to 100")
	}
	return limit, nil
}

func writeStoreError(writer http.ResponseWriter, err error) {
	if errors.Is(err, events.ErrInvalidCursor) {
		writeJSON(writer, http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
		return
	}
	writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "internal error"})
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
