package connections

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/o-mid/contract-ops/api/internal/credentials"
)

type Verifier interface {
	Verify(ctx context.Context, kind, secret string) error
}

// StaticVerifier accepts any non-empty fakevendor secret.
// OpenAI and Anthropic stay closed until their connectors are wired.
type StaticVerifier struct{}

func (StaticVerifier) Verify(_ context.Context, kind, secret string) error {
	switch kind {
	case "fakevendor":
		if strings.TrimSpace(secret) == "" {
			return &Failure{Status: 401, Code: "auth_invalid", Detail: "credential was rejected"}
		}
		return nil
	case "openai", "anthropic":
		return &Failure{Status: 400, Detail: "connector is not enabled"}
	default:
		return &Failure{Status: 400, Detail: "unknown connector"}
	}
}

type Failure struct {
	Status int
	Code   string
	Detail string
}

func (f *Failure) Error() string {
	return f.Detail
}

type Service struct {
	store    *Store
	sealer   *credentials.Sealer
	verifier Verifier
}

func NewService(store *Store, sealer *credentials.Sealer, verifier Verifier) *Service {
	return &Service{store: store, sealer: sealer, verifier: verifier}
}

func (s *Service) Create(ctx context.Context, workspaceID string, input CreateInput) (Connection, error) {
	input.Name = strings.TrimSpace(input.Name)
	if !validKind(input.Kind) {
		return Connection{}, &Failure{Status: 400, Detail: "unknown connector"}
	}
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 80 {
		return Connection{}, &Failure{Status: 400, Detail: "name must be 1 to 80 characters"}
	}
	if input.Secret == "" || len(input.Secret) > 4096 {
		return Connection{}, &Failure{Status: 400, Detail: "secret is required"}
	}

	sealed, err := s.sealer.Seal(ctx, []byte(input.Secret))
	if err != nil {
		return Connection{}, err
	}
	input.Secret = ""
	return s.store.Create(ctx, workspaceID, input, sealed)
}

func (s *Service) List(ctx context.Context, workspaceID string) ([]Connection, error) {
	return s.store.List(ctx, workspaceID)
}

func (s *Service) Get(ctx context.Context, workspaceID, id string) (Connection, error) {
	return s.store.Get(ctx, workspaceID, id)
}

func (s *Service) Rename(ctx context.Context, workspaceID, id, name string) (Connection, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 80 {
		return Connection{}, &Failure{Status: 400, Detail: "name must be 1 to 80 characters"}
	}
	return s.store.Rename(ctx, workspaceID, id, name)
}

func (s *Service) SetPaused(ctx context.Context, workspaceID, id string, paused bool) (Connection, error) {
	return s.store.SetPaused(ctx, workspaceID, id, paused)
}

func (s *Service) Verify(ctx context.Context, workspaceID, id string) (Connection, error) {
	kind, sealed, err := s.store.ActiveSecret(ctx, workspaceID, id)
	if err != nil {
		return Connection{}, err
	}
	secret, err := s.sealer.Open(ctx, sealed)
	if err != nil {
		return Connection{}, err
	}
	return s.finishVerify(ctx, workspaceID, id, kind, string(secret))
}

func (s *Service) Rotate(ctx context.Context, workspaceID, id string, input RotateInput) (Connection, error) {
	if input.Secret == "" || len(input.Secret) > 4096 {
		return Connection{}, &Failure{Status: 400, Detail: "secret is required"}
	}
	current, err := s.store.Get(ctx, workspaceID, id)
	if err != nil {
		return Connection{}, err
	}
	if err := s.verifier.Verify(ctx, current.Kind, input.Secret); err != nil {
		return Connection{}, err
	}

	sealed, err := s.sealer.Seal(ctx, []byte(input.Secret))
	if err != nil {
		return Connection{}, err
	}
	input.Secret = ""
	return s.store.Rotate(ctx, workspaceID, id, sealed, input.ExpiresAt)
}

func (s *Service) finishVerify(ctx context.Context, workspaceID, id, kind, secret string) (Connection, error) {
	err := s.verifier.Verify(ctx, kind, secret)
	var failure *Failure
	if err != nil {
		code := ""
		if errors.As(err, &failure) {
			code = failure.Code
		}
		next := StatusFailing
		if code != "" {
			next = Apply(StatusHealthy, code)
		}
		if _, statusErr := s.store.SetStatus(ctx, workspaceID, id, next, code, false); statusErr != nil {
			return Connection{}, statusErr
		}
		return Connection{}, err
	}
	return s.store.SetStatus(ctx, workspaceID, id, StatusHealthy, "", true)
}
