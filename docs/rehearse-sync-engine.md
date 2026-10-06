# Sync engine (60-second version)

1. **Enqueue** — scheduled or backfill jobs land in `sync_jobs` with a window. One running job per connection is enforced by a partial unique index plus an advisory lock at claim time.

2. **Claim** — workers poll with `FOR UPDATE SKIP LOCKED`, extend a lease (`lease_until`), and bump `attempt`. Stale leases can be stolen; a live lease on the same connection blocks a second runner.

3. **Fetch + normalize** — the connector pulls one vendor page for the window. Every record is normalized to FOCUS-shaped cost rows. Schema drift fails normalize and **quarantines** the batch: `raw_batches` is stored with `quarantined=true`, the job stops, and **`sync_cursors` is not advanced**.

4. **Commit** — on success, the batch hash is idempotent (`ON CONFLICT` on `connection_id + payload_hash`), cost rows are inserted in the same transaction, and the cursor high watermark moves forward. Retries of the same page do not double-insert costs.

5. **Rate limits** — vendor 429/5xx map to `Retry` with backoff and optional `Retry-After`. The job returns to `queued` without committing a batch, so there is no duplicate batch row and no cursor advance.

**Billing (P6)** — normalized rows are persisted in `cost_rows` and exposed at `GET /v1/costs`. OpenAI and Anthropic connectors verify keys and ingest organization cost payloads into the same pipeline as `fakevendor`.
