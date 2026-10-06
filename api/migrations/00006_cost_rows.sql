-- +goose Up
-- Normalized billing rows produced by sync jobs. Tied to the batch that ingested them.
CREATE TABLE cost_rows (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces (id),
    connection_id text NOT NULL REFERENCES connections (id),
    job_id text NOT NULL REFERENCES sync_jobs (id),
    batch_id text NOT NULL REFERENCES raw_batches (id),
    provider_name text NOT NULL,
    billing_account_id text NOT NULL DEFAULT '',
    service_name text NOT NULL,
    service_category text NOT NULL DEFAULT '',
    resource_id text NOT NULL DEFAULT '',
    sku_id text NOT NULL DEFAULT '',
    charge_category text NOT NULL DEFAULT '',
    charge_period_start timestamptz NOT NULL,
    charge_period_end timestamptz NOT NULL,
    billed_cost text NOT NULL,
    effective_cost text NOT NULL,
    billing_currency text NOT NULL DEFAULT 'USD',
    usage_quantity text NOT NULL DEFAULT '',
    usage_unit text NOT NULL DEFAULT '',
    source_record_id text NOT NULL,
    ingested_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT cost_rows_source_unique UNIQUE (connection_id, source_record_id, charge_period_start)
);

CREATE INDEX cost_rows_workspace_period_idx ON cost_rows (workspace_id, charge_period_start DESC);
CREATE INDEX cost_rows_connection_period_idx ON cost_rows (connection_id, charge_period_start DESC);

-- +goose Down
DROP TABLE cost_rows;
