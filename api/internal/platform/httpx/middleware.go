// Package httpx holds HTTP middleware shared by API routes.
package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

type requestIDKey struct{}

// RequestID copies a safe client-supplied id onto the context and response,
// or generates one. The id is what log lines use to tie a panic back to a call.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		id := request.Header.Get("X-Request-ID")
		if !validRequestID(id) {
			id = newRequestID()
		}
		writer.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(request.Context(), requestIDKey{}, id)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// IDFromContext returns the request id set by RequestID, or an empty string.
func IDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// AccessLog writes one JSON line after the handler returns.
func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			recorded := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
			next.ServeHTTP(recorded, request)
			logger.Info("request",
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.Int("status", recorded.status),
				slog.Int64("duration_ms", time.Since(started).Milliseconds()),
				slog.String("request_id", IDFromContext(request.Context())),
			)
		})
	}
}

// Recover turns a panic into a JSON 500 and a log line. The client never sees the stack.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				logger.Error("panic",
					slog.Any("error", rec),
					slog.String("request_id", IDFromContext(request.Context())),
					slog.String("path", request.URL.Path),
					slog.String("stack", string(debug.Stack())),
				)
				if tracker, ok := writer.(interface{ Written() bool }); ok && tracker.Written() {
					return
				}
				writer.Header().Set("Content-Type", "application/json; charset=utf-8")
				writer.WriteHeader(http.StatusInternalServerError)
				_, _ = writer.Write([]byte("{\"error\":\"internal error\"}\n"))
			}()
			next.ServeHTTP(writer, request)
		})
	}
}

// CORS sets the single allowed browser origin and answers preflight OPTIONS
// before auth middleware runs on mutating routes.
func CORS(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Access-Control-Allow-Origin", origin)
			writer.Header().Set("Vary", "Origin")
			if request.Method == http.MethodOptions {
				writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
				writer.Header().Set("Access-Control-Max-Age", "600")
				writer.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

// BodyLimit caps the request body. Handlers that never read a body are unaffected.
func BodyLimit(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.Body != nil {
				request.Body = http.MaxBytesReader(writer, request.Body, n)
			}
			next.ServeHTTP(writer, request)
		})
	}
}

// Timeout bounds ordinary requests. skip reports routes that must stay open,
// such as the events stream, which would otherwise be cut off mid-connection.
func Timeout(d time.Duration, skip func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		limited := http.TimeoutHandler(next, d, `{"error":"request timeout"}`)
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if skip != nil && skip(request) {
				next.ServeHTTP(writer, request)
				return
			}
			limited.ServeHTTP(writer, request)
		})
	}
}

func validRequestID(id string) bool {
	if len(id) == 0 || len(id) > 128 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func newRequestID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf[:])
}

type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (w *statusWriter) Written() bool {
	return w.wrote
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.status = status
	w.wrote = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.status = http.StatusOK
		w.wrote = true
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
