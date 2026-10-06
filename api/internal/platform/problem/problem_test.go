package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteIncludesCatalogAction(t *testing.T) {
	response := httptest.NewRecorder()
	Write(response, http.StatusUnauthorized, "auth_invalid", "rejected")

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("content type = %q", got)
	}

	var body Body
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "auth_invalid" || body.Action != "rotate_key" || body.Detail != "rejected" {
		t.Fatalf("body = %+v", body)
	}
	if body.Retryable == nil || *body.Retryable {
		t.Fatalf("retryable = %v", body.Retryable)
	}
}
