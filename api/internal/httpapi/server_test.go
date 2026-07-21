package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/o-mid/contract-ops/api/internal/events"
)

func TestListEventsReturnsFilteredEvents(t *testing.T) {
	server := NewServer(events.NewStore(events.Fixtures()))
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
	server := NewServer(events.NewStore(events.Fixtures()))
	request := httptest.NewRequest(http.MethodGet, "/v1/events?status=unknown", nil)
	response := httptest.NewRecorder()

	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}
