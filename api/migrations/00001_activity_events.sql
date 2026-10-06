-- +goose Up
-- activity_events is the activity feed behind GET /v1/events.
-- id is the source record id, so a replayed fixture or a retried insert is a no-op.
CREATE TABLE activity_events (
    id text PRIMARY KEY,
    source text NOT NULL,
    type text NOT NULL,
    status text NOT NULL,
    occurred_at timestamptz NOT NULL,
    correlation_id text NOT NULL,
    connection_id text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT activity_events_status_chk CHECK (status IN ('processed', 'pending', 'failed'))
);

-- Newest-first keyset pages filter and sort on (occurred_at, id).
CREATE INDEX activity_events_list_idx ON activity_events (occurred_at DESC, id DESC);

-- connection_id is empty for the demo feed. A partial index stays small until connections write rows.
CREATE INDEX activity_events_connection_idx ON activity_events (connection_id) WHERE connection_id IS NOT NULL;

-- +goose Down
DROP TABLE activity_events;
