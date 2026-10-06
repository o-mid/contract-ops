package httpapi

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/o-mid/contract-ops/api/internal/events"
)

func newTestServer(store *events.Store) Server {
	return NewServer(store, Options{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func TestListEventsReturnsFilteredEvents(t *testing.T) {
	server := newTestServer(events.NewStore(events.Fixtures()))
	request := httptest.NewRequest(http.MethodGet, "/v1/events?status=failed", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	expected := `{"events":[{"id":"evt_01HV1B03"`
	if got := response.Body.String(); len(got) < len(expected) || got[:len(expected)] != expected {
		t.Fatalf("expected filtered event response, got %q", got)
	}
}

func TestListEventsRejectsUnknownStatus(t *testing.T) {
	server := newTestServer(events.NewStore(events.Fixtures()))
	request := httptest.NewRequest(http.MethodGet, "/v1/events?status=unknown", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestStreamEventsRejectsUnknownStatus(t *testing.T) {
	server := newTestServer(events.NewStore(events.Fixtures()))
	request := httptest.NewRequest(http.MethodGet, "/v1/events/stream?status=unknown", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestStreamEventsSendsSnapshotAndUpdate(t *testing.T) {
	store := events.NewStore(events.Fixtures())
	server := newTestServer(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/events/stream?status=all", nil)
	response := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(response, request)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(response.Body.String(), `"total":4`) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(response.Body.String(), `"total":4`) {
		t.Fatalf("expected initial snapshot, got %q", response.Body.String())
	}

	pending := events.PendingFixtures()[0]
	pending.OccurredAt = time.Now().UTC()
	inserted, err := store.Append(context.Background(), pending)
	if err != nil || !inserted {
		t.Fatal("expected append to succeed")
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(response.Body.String(), pending.ID) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(response.Body.String(), pending.ID) {
		t.Fatalf("expected stream update with %q, got %q", pending.ID, response.Body.String())
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stream handler did not exit after cancel")
	}

	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("expected event-stream content type, got %q", contentType)
	}

	scanner := bufio.NewScanner(strings.NewReader(response.Body.String()))
	sawData := false
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "data: ") {
			sawData = true
			break
		}
	}
	if !sawData {
		t.Fatal("expected SSE data lines")
	}
}
