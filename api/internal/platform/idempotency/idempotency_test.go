package idempotency

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/o-mid/contract-ops/api/internal/platform/auth"
)

func TestReplayReturnsTheFirstResponse(t *testing.T) {
	keeper := newMemory()
	calls := 0
	handler := Middleware(keeper)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		calls++
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(writer, `{"id":"conn_1"}`)
	}))

	first := do(t, handler, `{"name":"a"}`)
	second := do(t, handler, `{"name":"a"}`)

	if calls != 1 {
		t.Fatalf("handler calls = %d", calls)
	}
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("statuses = %d %d", first.Code, second.Code)
	}
	if second.Body.String() != `{"id":"conn_1"}` {
		t.Fatalf("replay body = %s", second.Body.String())
	}
}

func TestDifferentBodyConflicts(t *testing.T) {
	handler := Middleware(newMemory())(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
	}))

	first := do(t, handler, `{"name":"a"}`)
	second := do(t, handler, `{"name":"b"}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("first status = %d", first.Code)
	}
	if second.Code != http.StatusConflict {
		t.Fatalf("second status = %d, body %s", second.Code, second.Body.String())
	}
}

func TestInProgressKeyConflicts(t *testing.T) {
	handler := Middleware(holding{})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not run")
	}))
	response := do(t, handler, `{"name":"a"}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d", response.Code)
	}
}

type holding struct{}

func (holding) Claim(context.Context, string, string, []byte) (Record, bool, error) {
	return Record{}, false, nil
}

func (holding) Complete(context.Context, string, string, int, []byte, string) error {
	return nil
}

func (holding) Release(context.Context, string, string) error {
	return nil
}

func TestGetIgnoresTheKey(t *testing.T) {
	calls := 0
	handler := Middleware(newMemory())(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		calls++
		writer.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	request.Header.Set(headerKey, "same")
	request = request.WithContext(auth.WithWorkspace(request.Context(), "ws_1"))
	handler.ServeHTTP(httptest.NewRecorder(), request)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
}

func do(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/v1/connections", strings.NewReader(body))
	request.Header.Set(headerKey, "req_1")
	request = request.WithContext(auth.WithWorkspace(request.Context(), "ws_1"))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type memory struct {
	mu   sync.Mutex
	rows map[string]Record
}

func newMemory() *memory {
	return &memory{rows: map[string]Record{}}
}

func (m *memory) Claim(_ context.Context, workspaceID, key string, requestHash []byte) (Record, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := workspaceID + "\x00" + key
	if existing, ok := m.rows[id]; ok {
		return existing, false, nil
	}
	record := Record{RequestHash: append([]byte(nil), requestHash...)}
	m.rows[id] = record
	return record, true, nil
}

func (m *memory) Complete(_ context.Context, workspaceID, key string, status int, body []byte, contentType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := workspaceID + "\x00" + key
	record := m.rows[id]
	record.Status = status
	record.Body = append([]byte(nil), body...)
	record.ContentType = contentType
	m.rows[id] = record
	return nil
}

func (m *memory) Release(_ context.Context, workspaceID, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rows, workspaceID+"\x00"+key)
	return nil
}
