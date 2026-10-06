// Package auth checks workspace API keys. The secret is hashed; only the hash and a short prefix are stored.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/o-mid/contract-ops/api/internal/platform/problem"
)

const LocalWorkspaceID = "ws_local"

var ErrUnknownKey = errors.New("unknown api key")

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func Hash(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

func Prefix(secret string) string {
	if len(secret) <= 8 {
		return secret
	}
	return secret[:8]
}

// EnsureBootstrap inserts the local workspace and key when a secret is configured.
// An empty secret leaves the database unchanged. Repeating the same secret is a no-op.
func (s *Store) EnsureBootstrap(ctx context.Context, secret string) error {
	if secret == "" {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `INSERT INTO workspaces (id, name) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`, LocalWorkspaceID, "Local"); err != nil {
		return err
	}

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM api_keys WHERE key_hash = $1 AND revoked_at IS NULL)`, Hash(secret)).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return tx.Commit(ctx)
	}

	id, err := newID("key_")
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO api_keys (id, workspace_id, prefix, key_hash) VALUES ($1, $2, $3, $4)`, id, LocalWorkspaceID, Prefix(secret), Hash(secret)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Resolve(ctx context.Context, secret string) (string, error) {
	var workspaceID string
	err := s.pool.QueryRow(ctx, `SELECT workspace_id FROM api_keys WHERE key_hash = $1 AND revoked_at IS NULL`, Hash(secret)).Scan(&workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUnknownKey
	}
	if err != nil {
		return "", err
	}
	return workspaceID, nil
}

type contextKey struct{}

func WorkspaceID(ctx context.Context) string {
	id, _ := ctx.Value(contextKey{}).(string)
	return id
}

func WithWorkspace(ctx context.Context, workspaceID string) context.Context {
	return context.WithValue(ctx, contextKey{}, workspaceID)
}

type Resolver func(ctx context.Context, secret string) (workspaceID string, err error)

// Middleware requires a bearer key on routes that are not public.
// The event feed is public because the browser EventSource cannot set Authorization.
func Middleware(resolve Resolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if public(request.URL.Path) {
				next.ServeHTTP(writer, request)
				return
			}

			secret, ok := bearer(request.Header.Get("Authorization"))
			if !ok {
				problem.Write(writer, http.StatusUnauthorized, "auth_invalid", "missing bearer token")
				return
			}

			workspaceID, err := resolve(request.Context(), secret)
			if errors.Is(err, ErrUnknownKey) {
				problem.Write(writer, http.StatusUnauthorized, "auth_invalid", "unknown api key")
				return
			}
			if err != nil {
				problem.Write(writer, http.StatusInternalServerError, "", "could not check api key")
				return
			}

			next.ServeHTTP(writer, request.WithContext(WithWorkspace(request.Context(), workspaceID)))
		})
	}
}

func public(path string) bool {
	switch path {
	case "/healthz", "/readyz", "/metrics", "/v1/events", "/v1/events/stream":
		return true
	default:
		return false
	}
}

func bearer(header string) (string, bool) {
	scheme, secret, ok := strings.Cut(header, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", false
	}
	return secret, true
}

func newID(prefix string) (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf[:]), nil
}
