package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoRetries429ThenReturns200(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			writer.Header().Set("Retry-After", "1")
			writer.WriteHeader(http.StatusTooManyRequests)
			return
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	var waited time.Duration
	client := Client{
		HTTP:        server.Client(),
		MaxAttempts: 2,
		Sleep: func(_ context.Context, wait time.Duration) error {
			waited = wait
			return nil
		},
	}
	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { Discard(response.Body) })
	if response.StatusCode != http.StatusOK || calls.Load() != 2 {
		t.Fatalf("status %d calls %d", response.StatusCode, calls.Load())
	}
	if waited < time.Second {
		t.Fatalf("waited %s, want at least 1s", waited)
	}
}

func TestDoStopsAfterTheAttemptCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := Client{
		HTTP:        server.Client(),
		MaxAttempts: 2,
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}
	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Do(context.Background(), request); err == nil {
		t.Fatal("expected the attempt cap to fail")
	}
}
