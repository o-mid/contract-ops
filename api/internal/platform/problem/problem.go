// Package problem writes RFC 9457 problem details. The code field matches the shared catalog.
package problem

import (
	"encoding/json"
	"net/http"

	"github.com/o-mid/contract-ops/api/internal/platform/catalog"
)

type Body struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Code      string `json:"code,omitempty"`
	Retryable *bool  `json:"retryable,omitempty"`
	Action    string `json:"action,omitempty"`
}

// Write sends application/problem+json. An unknown code is still returned so
// the client can branch on it, without inventing catalog metadata.
func Write(writer http.ResponseWriter, status int, code, detail string) {
	body := Body{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
		Code:   code,
	}
	if entry, ok := catalog.ByCode[code]; ok {
		retryable := entry.Retryable
		body.Retryable = &retryable
		body.Action = entry.Action
	}

	writer.Header().Set("Content-Type", "application/problem+json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(body)
}
