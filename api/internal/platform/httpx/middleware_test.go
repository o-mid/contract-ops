package httpx

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestIDUsesASafeClientValue(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := IDFromContext(request.Context()); got != "abc-123" {
			t.Fatalf("context id = %q", got)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("X-Request-ID", "abc-123")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("X-Request-ID"); got != "abc-123" {
		t.Fatalf("response id = %q", got)
	}
}

func TestRequestIDReplacesUnsafeClientValue(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("X-Request-ID", "bad id\n")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	got := response.Header().Get("X-Request-ID")
	if got == "" || strings.Contains(got, "bad") {
		t.Fatalf("expected a generated id, got %q", got)
	}
}

func TestRecoverWritesJSONAndKeepsGoing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), `"error":"internal error"`) {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestCORSSetsConfiguredOrigin(t *testing.T) {
	handler := CORS("http://localhost:5173")(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("origin = %q", got)
	}
	if got := response.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("vary = %q", got)
	}
}

func TestTimeoutSkipsLongLivedRoutes(t *testing.T) {
	handler := Timeout(time.Second, func(request *http.Request) bool {
		return request.URL.Path == "/v1/events/stream"
	})(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, hasDeadline := request.Context().Deadline()
		if request.URL.Path == "/v1/events/stream" && hasDeadline {
			t.Error("stream request should not carry the timeout deadline")
		}
		if request.URL.Path == "/healthz" && !hasDeadline {
			t.Error("ordinary request should carry the timeout deadline")
		}
		writer.WriteHeader(http.StatusNoContent)
	}))

	for _, path := range []string{"/healthz", "/v1/events/stream"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s status = %d", path, response.Code)
		}
	}
}

func TestTimeoutSkipSeesParentContextCancel(t *testing.T) {
	handler := Timeout(time.Hour, func(request *http.Request) bool {
		return true
	})(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
		if request.Context().Err() != context.Canceled {
			t.Fatalf("err = %v", request.Context().Err())
		}
		writer.WriteHeader(http.StatusNoContent)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/events/stream", nil)
	response := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(response, request)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("skipped route did not observe cancellation")
	}
}
