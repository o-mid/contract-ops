package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMiddlewareLeavesTheEventFeedOpen(t *testing.T) {
	router := testRouter(func(context.Context, string) (string, error) {
		return "", ErrUnknownKey
	})

	for _, path := range []string{"/healthz", "/v1/events", "/v1/events/stream"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, response.Code)
		}
	}
}

func TestMiddlewareRequiresAKnownKeyOnOtherRoutes(t *testing.T) {
	const secret = "co_known_secret_value_ok"
	router := testRouter(func(_ context.Context, got string) (string, error) {
		if got != secret {
			return "", ErrUnknownKey
		}
		return "ws_1", nil
	})

	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "/v1/connections", nil))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing status = %d", missing.Code)
	}

	rejected := httptest.NewRecorder()
	bad := httptest.NewRequest(http.MethodPost, "/v1/connections", nil)
	bad.Header.Set("Authorization", "Bearer wrong-key-value")
	router.ServeHTTP(rejected, bad)
	if rejected.Code != http.StatusUnauthorized {
		t.Fatalf("rejected status = %d", rejected.Code)
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/connections", nil)
	request.Header.Set("Authorization", "Bearer "+secret)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("accepted status = %d, body %s", response.Code, response.Body.String())
	}
}

func testRouter(resolve Resolver) http.Handler {
	router := chi.NewRouter()
	router.Use(Middleware(resolve))
	router.Get("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	router.Get("/v1/events", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	router.Get("/v1/events/stream", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	router.Post("/v1/connections", func(writer http.ResponseWriter, request *http.Request) {
		if WorkspaceID(request.Context()) != "ws_1" {
			http.Error(writer, "missing workspace", http.StatusInternalServerError)
			return
		}
		writer.WriteHeader(http.StatusCreated)
	})
	return router
}
