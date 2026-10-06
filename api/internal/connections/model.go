package connections

import "time"

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusFailing   Status = "failing"
	StatusNeedsAuth Status = "needs_auth"
	StatusPaused    Status = "paused"
)

type Connection struct {
	ID               string          `json:"id"`
	Kind             string          `json:"kind"`
	Name             string          `json:"name"`
	Status           Status          `json:"status"`
	StatusReasonCode string          `json:"statusReasonCode,omitempty"`
	LastSuccessAt    *time.Time      `json:"lastSuccessAt,omitempty"`
	LastErrorAt      *time.Time      `json:"lastErrorAt,omitempty"`
	Credential       *CredentialView `json:"credential,omitempty"`
}

type CredentialView struct {
	Fingerprint string     `json:"fingerprint"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
}

type CreateInput struct {
	Kind      string     `json:"kind"`
	Name      string     `json:"name"`
	Secret    string     `json:"secret"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

type RotateInput struct {
	Secret    string     `json:"secret"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

func validKind(kind string) bool {
	switch kind {
	case "fakevendor", "openai", "anthropic":
		return true
	default:
		return false
	}
}
