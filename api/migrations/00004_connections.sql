-- +goose Up
-- A connection is one workspace's link to a vendor. Status is the health
-- graph: healthy, degraded, failing, needs_auth, paused.
CREATE TABLE connections (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces (id),
    kind text NOT NULL,
    name text NOT NULL,
    status text NOT NULL,
    status_reason_code text NOT NULL DEFAULT '',
    paused_from text,
    last_success_at timestamptz,
    last_error_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT connections_status_chk CHECK (status IN ('healthy', 'degraded', 'failing', 'needs_auth', 'paused')),
    CONSTRAINT connections_kind_chk CHECK (kind IN ('fakevendor', 'openai', 'anthropic'))
);

-- The console lists one workspace's connections, newest first.
CREATE INDEX connections_workspace_idx ON connections (workspace_id, created_at DESC);

-- The live credential is the row with rotated_at IS NULL. Older rows stay
-- so a rotation can be checked without ever returning the secret.
CREATE TABLE credentials (
    id text PRIMARY KEY,
    connection_id text NOT NULL REFERENCES connections (id),
    ciphertext bytea NOT NULL,
    wrapped_key bytea NOT NULL,
    key_version integer NOT NULL,
    fingerprint text NOT NULL,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    rotated_at timestamptz
);

CREATE UNIQUE INDEX credentials_active_idx ON credentials (connection_id) WHERE rotated_at IS NULL;

-- +goose Down
DROP TABLE credentials;
DROP TABLE connections;
