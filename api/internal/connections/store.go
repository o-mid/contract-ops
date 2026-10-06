package connections

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/o-mid/contract-ops/api/internal/credentials"
	"github.com/o-mid/contract-ops/api/internal/events"
)

var ErrNotFound = errors.New("connection not found")

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, workspaceID string, input CreateInput, sealed credentials.Sealed) (Connection, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Connection{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	connID, err := newID("conn_")
	if err != nil {
		return Connection{}, err
	}
	credID, err := newID("cred_")
	if err != nil {
		return Connection{}, err
	}
	now := time.Now().UTC()

	if _, err := tx.Exec(ctx, `
		INSERT INTO connections (id, workspace_id, kind, name, status, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		connID, workspaceID, input.Kind, input.Name, string(StatusNeedsAuth), now); err != nil {
		return Connection{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO credentials (id, connection_id, ciphertext, wrapped_key, key_version, fingerprint, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		credID, connID, sealed.Ciphertext, sealed.WrappedKey, sealed.KeyVersion, sealed.Fingerprint, input.ExpiresAt); err != nil {
		return Connection{}, err
	}
	if err := insertEvent(ctx, tx, connID, "connection.created", now); err != nil {
		return Connection{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Connection{}, err
	}

	return Connection{
		ID:     connID,
		Kind:   input.Kind,
		Name:   input.Name,
		Status: StatusNeedsAuth,
		Credential: &CredentialView{
			Fingerprint: sealed.Fingerprint,
			ExpiresAt:   input.ExpiresAt,
		},
	}, nil
}

func (s *Store) List(ctx context.Context, workspaceID string) ([]Connection, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.kind, c.name, c.status, c.status_reason_code, c.last_success_at, c.last_error_at,
		       cred.fingerprint, cred.expires_at
		FROM connections c
		LEFT JOIN credentials cred ON cred.connection_id = c.id AND cred.rotated_at IS NULL
		WHERE c.workspace_id = $1
		ORDER BY c.created_at DESC, c.id DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Connection
	for rows.Next() {
		conn, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, conn)
	}
	return out, rows.Err()
}

func (s *Store) Get(ctx context.Context, workspaceID, id string) (Connection, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT c.id, c.kind, c.name, c.status, c.status_reason_code, c.last_success_at, c.last_error_at,
		       cred.fingerprint, cred.expires_at
		FROM connections c
		LEFT JOIN credentials cred ON cred.connection_id = c.id AND cred.rotated_at IS NULL
		WHERE c.workspace_id = $1 AND c.id = $2`, workspaceID, id)
	conn, err := scanConnection(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Connection{}, ErrNotFound
	}
	return conn, err
}

func (s *Store) ActiveSecret(ctx context.Context, workspaceID, id string) (string, credentials.Sealed, error) {
	var kind string
	var sealed credentials.Sealed
	err := s.pool.QueryRow(ctx, `
		SELECT c.kind, cred.ciphertext, cred.wrapped_key, cred.key_version, cred.fingerprint
		FROM connections c
		JOIN credentials cred ON cred.connection_id = c.id AND cred.rotated_at IS NULL
		WHERE c.workspace_id = $1 AND c.id = $2`, workspaceID, id).Scan(
		&kind, &sealed.Ciphertext, &sealed.WrappedKey, &sealed.KeyVersion, &sealed.Fingerprint)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", credentials.Sealed{}, ErrNotFound
	}
	if err != nil {
		return "", credentials.Sealed{}, err
	}
	return kind, sealed, nil
}

func (s *Store) SetStatus(ctx context.Context, workspaceID, id string, status Status, reason string, success bool) (Connection, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Connection{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()
	var successAt, errorAt any
	if success {
		successAt = now
	}
	if reason != "" {
		errorAt = now
	}
	tag, err := tx.Exec(ctx, `
		UPDATE connections
		SET status = $3,
		    status_reason_code = $4,
		    last_success_at = COALESCE($5, last_success_at),
		    last_error_at = COALESCE($6, last_error_at),
		    updated_at = $7
		WHERE workspace_id = $1 AND id = $2 AND status <> 'paused'`,
		workspaceID, id, string(status), reason, successAt, errorAt, now)
	if err != nil {
		return Connection{}, err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM connections WHERE workspace_id = $1 AND id = $2)`, workspaceID, id).Scan(&exists); err != nil {
			return Connection{}, err
		}
		if !exists {
			return Connection{}, ErrNotFound
		}
	} else if err := insertEvent(ctx, tx, id, "connection.status_changed", now); err != nil {
		return Connection{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Connection{}, err
	}
	return s.Get(ctx, workspaceID, id)
}

func (s *Store) SetPaused(ctx context.Context, workspaceID, id string, paused bool) (Connection, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Connection{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()
	var query string
	var eventType string
	if paused {
		eventType = "connection.paused"
		query = `
			UPDATE connections
			SET paused_from = status, status = 'paused', updated_at = $3
			WHERE workspace_id = $1 AND id = $2 AND status <> 'paused'`
	} else {
		eventType = "connection.resumed"
		query = `
			UPDATE connections
			SET status = COALESCE(paused_from, 'needs_auth'), paused_from = NULL, updated_at = $3
			WHERE workspace_id = $1 AND id = $2 AND status = 'paused'`
	}
	tag, err := tx.Exec(ctx, query, workspaceID, id, now)
	if err != nil {
		return Connection{}, err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM connections WHERE workspace_id = $1 AND id = $2)`, workspaceID, id).Scan(&exists); err != nil {
			return Connection{}, err
		}
		if !exists {
			return Connection{}, ErrNotFound
		}
	} else if err := insertEvent(ctx, tx, id, eventType, now); err != nil {
		return Connection{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Connection{}, err
	}
	return s.Get(ctx, workspaceID, id)
}

func (s *Store) Rotate(ctx context.Context, workspaceID, id string, sealed credentials.Sealed, expiresAt *time.Time) (Connection, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Connection{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM connections WHERE workspace_id = $1 AND id = $2)`, workspaceID, id).Scan(&exists); err != nil {
		return Connection{}, err
	}
	if !exists {
		return Connection{}, ErrNotFound
	}

	now := time.Now().UTC()
	credID, err := newID("cred_")
	if err != nil {
		return Connection{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE credentials SET rotated_at = $2 WHERE connection_id = $1 AND rotated_at IS NULL`, id, now); err != nil {
		return Connection{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO credentials (id, connection_id, ciphertext, wrapped_key, key_version, fingerprint, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		credID, id, sealed.Ciphertext, sealed.WrappedKey, sealed.KeyVersion, sealed.Fingerprint, expiresAt); err != nil {
		return Connection{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE connections
		SET status = 'healthy', status_reason_code = '', last_success_at = $3, updated_at = $3
		WHERE id = $1 AND workspace_id = $2 AND status <> 'paused'`,
		id, workspaceID, now); err != nil {
		return Connection{}, err
	}
	if err := insertEvent(ctx, tx, id, "connection.rotated", now); err != nil {
		return Connection{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Connection{}, err
	}
	return s.Get(ctx, workspaceID, id)
}

func (s *Store) Rename(ctx context.Context, workspaceID, id, name string) (Connection, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE connections SET name = $3, updated_at = $4 WHERE workspace_id = $1 AND id = $2`, workspaceID, id, name, time.Now().UTC())
	if err != nil {
		return Connection{}, err
	}
	if tag.RowsAffected() == 0 {
		return Connection{}, ErrNotFound
	}
	return s.Get(ctx, workspaceID, id)
}

type scannable interface {
	Scan(dest ...any) error
}

func scanConnection(row scannable) (Connection, error) {
	var conn Connection
	var status string
	var fingerprint *string
	var expires *time.Time
	if err := row.Scan(&conn.ID, &conn.Kind, &conn.Name, &status, &conn.StatusReasonCode, &conn.LastSuccessAt, &conn.LastErrorAt, &fingerprint, &expires); err != nil {
		return Connection{}, err
	}
	conn.Status = Status(status)
	if fingerprint != nil {
		conn.Credential = &CredentialView{Fingerprint: *fingerprint, ExpiresAt: expires}
	}
	return conn, nil
}

func insertEvent(ctx context.Context, tx pgx.Tx, connectionID, eventType string, at time.Time) error {
	id, err := newID("evt_")
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO activity_events (id, source, type, status, occurred_at, correlation_id, connection_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, "connections", eventType, string(events.StatusProcessed), at, connectionID, connectionID)
	return err
}

func newID(prefix string) (string, error) {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf[:]), nil
}
