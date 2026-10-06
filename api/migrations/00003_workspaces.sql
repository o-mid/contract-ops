-- +goose Up
-- A workspace owns connections and API keys. The local process seeds one row
-- from BOOTSTRAP_API_KEY; there is no user table.
CREATE TABLE workspaces (
    id text PRIMARY KEY,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- key_hash is sha256 of the secret. The secret itself is never stored.
-- prefix is the non-secret start of the key, for display in the console.
CREATE TABLE api_keys (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces (id),
    prefix text NOT NULL,
    key_hash bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz
);

-- A live key is found by hash. Revoked keys stay in the table and must not match.
CREATE UNIQUE INDEX api_keys_hash_idx ON api_keys (key_hash) WHERE revoked_at IS NULL;

-- The console lists keys for one workspace.
CREATE INDEX api_keys_workspace_idx ON api_keys (workspace_id);

-- One stored response per workspace and idempotency key. status_code 0 means
-- a request claimed the key and has not finished.
CREATE TABLE idempotency_keys (
    workspace_id text NOT NULL REFERENCES workspaces (id),
    idempotency_key text NOT NULL,
    request_hash bytea NOT NULL,
    status_code integer NOT NULL,
    response_body bytea NOT NULL,
    content_type text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, idempotency_key)
);

-- +goose Down
DROP TABLE idempotency_keys;
DROP TABLE api_keys;
DROP TABLE workspaces;
