-- +goose Up
-- Jobs are the unit of sync work. A running row holds a lease so a crashed
-- worker can be replaced, and only one running job is allowed per connection.
-- Chunks for one connection therefore run one after another. Different
-- connections can run at the same time.
CREATE TABLE sync_jobs (
    id text PRIMARY KEY,
    connection_id text NOT NULL REFERENCES connections (id),
    workspace_id text NOT NULL REFERENCES workspaces (id),
    kind text NOT NULL,
    window_start timestamptz NOT NULL,
    window_end timestamptz NOT NULL,
    status text NOT NULL,
    attempt integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 5,
    next_run_at timestamptz NOT NULL DEFAULT now(),
    lease_until timestamptz,
    heartbeat_at timestamptz,
    error_code text NOT NULL DEFAULT '',
    error_detail text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sync_jobs_kind_chk CHECK (kind IN ('scheduled', 'backfill', 'reconnect')),
    CONSTRAINT sync_jobs_status_chk CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled', 'quarantined'))
);

-- The claim query filters on due time and skips locked rows.
CREATE INDEX sync_jobs_claim_idx ON sync_jobs (next_run_at, id)
    WHERE status IN ('queued', 'running');

-- One leased job per connection. Parallel chunks of the same connection would
-- upsert the same lookback window.
CREATE UNIQUE INDEX sync_jobs_one_running_idx ON sync_jobs (connection_id)
    WHERE status = 'running';

-- The scheduler must not enqueue a second scheduled job while one is active.
CREATE UNIQUE INDEX sync_jobs_active_scheduled_idx ON sync_jobs (connection_id)
    WHERE kind = 'scheduled' AND status IN ('queued', 'running');

-- A repeated backfill of the same window joins the job that is already queued.
CREATE UNIQUE INDEX sync_jobs_window_idx ON sync_jobs (connection_id, kind, window_start, window_end)
    WHERE status IN ('queued', 'running');

-- The high watermark is the newest page committed for a stream. Drift leaves
-- this row unchanged.
CREATE TABLE sync_cursors (
    connection_id text NOT NULL REFERENCES connections (id),
    stream text NOT NULL,
    cursor text NOT NULL DEFAULT '',
    high_watermark timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (connection_id, stream)
);

-- payload_hash includes the window, so a retry of the same page is one row
-- and a later day is a different row.
CREATE TABLE raw_batches (
    id text PRIMARY KEY,
    job_id text NOT NULL REFERENCES sync_jobs (id),
    connection_id text NOT NULL REFERENCES connections (id),
    received_at timestamptz NOT NULL DEFAULT now(),
    record_count integer NOT NULL,
    payload_hash bytea NOT NULL,
    quarantined boolean NOT NULL DEFAULT false,
    drift_report jsonb,
    CONSTRAINT raw_batches_hash_unique UNIQUE (connection_id, payload_hash)
);

-- +goose Down
DROP TABLE raw_batches;
DROP TABLE sync_cursors;
DROP TABLE sync_jobs;
