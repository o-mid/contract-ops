# Architecture

Contract Ops has two processes that share one Postgres database, and a static console that only reads the activity feed.

The API accepts HTTP, seals credentials, and records work. The worker is the only process that calls a connector's `Fetch`. Several API replicas can serve the feed. Several workers can claim jobs. One scheduler, chosen with a Postgres advisory lock, is allowed to enqueue the periodic jobs.

```text
browser
  EventSource  GET /v1/events/stream
       |
       v
api  ---- activity_events ---- LISTEN/NOTIFY (empty payload)
       |
       +---- connections, credentials, sync_jobs
                |
                v
worker  claims a job, opens the secret, Fetch, Normalize
                |
                +---- raw_batches + sync_cursors, or a quarantined batch
```

## Boot

`cmd/api` and `cmd/worker` both call goose before they open their pool. That is how a single container can migrate: the image entrypoint is `/api`, so a platform start command cannot chain `/migrate` and then `/api`. Compose still runs `cmd/migrate` as its own service and waits until it exits. Running `Up` again is a no-op once the schema matches.

`/readyz` pings the pool and compares `max(version_id)` in `goose_db_version` with the highest numeric prefix embedded in the binary. A process that is behind the schema fails ready and should not take traffic. The expected version is read from the embedded files. It is not a hardcoded number in the ready check. The ready test currently expects 5, and that assertion has to move when a migration is added.

The pool allows 10 connections, idle for up to 5 minutes, with a health check every 30 seconds. Each connection sets `statement_timeout` to 5 seconds. Open pings before it returns.

Config is environment variables, checked before the process binds. `PORT` defaults to 8080 and must be 1 through 65535. `DATABASE_URL` must be a `postgres` or `postgresql` URL. `MASTER_KEY` is required for the API and the worker: base64 of exactly 32 bytes. `BOOTSTRAP_API_KEY`, when set, must be at least 20 characters and contain no whitespace. `DEMO_GENERATOR` defaults to true.

Shutdown on SIGINT or SIGTERM waits `SHUTDOWN_TIMEOUT` (default 5 seconds) for the HTTP server. The LISTEN loop is stopped before the pool closes: `stopListen` is deferred after `pool.Close`, and defers run in reverse, so the listener finishes while the pool is still open.

## HTTP

chi middleware, outer to inner:

1. Request id. A safe client `X-Request-ID` is kept. Otherwise one is generated.
2. CORS, one origin, default `http://localhost:5173`.
3. Access log, one JSON line after the handler returns.
4. Recover.
5. Prometheus instrumentation, when wired. The recorder implements `Flush` and `Unwrap` so an SSE write still flushes.
6. Body limit, 1 MiB.
7. Auth, when wired.
8. Idempotency, when wired.
9. Timeout, 30 seconds, except `GET /v1/events/stream`. A timeout there would look like a broken stream to the console. The browser closes the stream.

`ReadHeaderTimeout` on the server is 5 seconds. Logs are JSON on stdout.

`/metrics` uses a private Prometheus registry, not the global one, so tests do not share counters. The series are `http_requests_total` and `http_request_duration_seconds`, labelled with the chi route pattern.

## Activity feed

`activity_events` is the feed behind `GET /v1/events`. The id is the source record id. Inserting a fixture that is already there is a no-op (`ON CONFLICT DO NOTHING`).

Pages are a keyset on `(occurred_at, id)`, newest first. The cursor is base64, raw URL encoding, of an RFC3339Nano timestamp, a newline, and the id. The next page is rows strictly older than that pair. `total` is the filtered count, not the page length. `nextCursor` is omitted when the page is the last one.

`q` uses `strpos` on the lowercased id, source, type, and correlation id.

A statement-level trigger notifies channel `activity_events` with an empty payload. The notice is a wake-up. Subscribers re-read the page, so a dropped or oversized notification cannot publish a stale snapshot. Inserts in the same process also wake local subscribers directly. Two API processes prove the path: each holds a `LISTEN`, and a write in one is visible to the other. Compose does not scale the API, because both would want port 8080.

The demo generator appends the pending fixtures every 5 seconds and stops at `MaxEvents` (8). Duplicate ids do not count as a new row.

`internal/events.Store` is the in-memory feed. Unit tests use it. The running API uses `activity.Store`, which implements the same `Feed` interface.

The console opens one `EventSource` per filter change. `useDeferredValue` keeps typing from reconnecting on every key. The stream payload is checked at runtime: `total` has to be a finite non-negative number, `events` an array, and each status one of `processed`, `pending`, `failed`. Extra fields, including `nextCursor`, are ignored. The console renders the snapshot it was given. It does not follow the cursor. Loading is shown when the filters change, not on every SSE tick.

## Auth and idempotency

Public paths are `/healthz`, `/readyz`, `/metrics`, `/v1/events`, and `/v1/events/stream`. Anything else needs a bearer token.

The token is hashed with SHA-256. The hash is what `api_keys` stores, plus the first 8 characters as a prefix for display. A unique index on the hash covers rows where `revoked_at` is null. `BOOTSTRAP_API_KEY` seeds workspace `ws_local`, named "Local", when that hash is not already present. There is no user table.

A resolved key puts the workspace id on the request context. Handlers read it with `auth.WorkspaceID`.

Idempotency applies to `POST`, `PUT`, and `PATCH` when a workspace is present and the client sent `Idempotency-Key`. The stored hash is method, path, query, and body. Claim inserts a row with status 0, meaning in progress. The handler's response is buffered, then stored by `Complete`, then written. The same key and body replay that response. A different body returns 409. A second request that arrives while status is still 0 returns 409. `Release` drops the claim when the handler fails before it can complete.

Problem responses are `application/problem+json`. `api/openapi/errors.yaml` is the catalog. `cmd/gencatalog` writes `internal/platform/catalog/catalog.gen.go` and `src/domain/errorCatalog.gen.ts`. CI runs `gencatalog -check`. An unknown code still returns a problem body. It just has no catalog metadata. The generated TypeScript catalog is not imported by the console yet.

`oapi-codegen` generates models only, into `internal/platform/apigen/types.gen.go`. Handlers are written by hand. CI regenerates that file and fails on a diff.

## Connections and credentials

A connection belongs to one workspace. The kind check allows `fakevendor`, `openai`, and `anthropic`. Status is `healthy`, `degraded`, `failing`, `needs_auth`, or `paused`.

Create starts at `needs_auth`. The secret is sealed, then cleared from the in-memory input, then stored. Verify calls the connector. Success moves the connection to `healthy`.

Rotate verifies the new secret first. Only then does it seal and swap. A rejected secret leaves the previous row active. The active credential is the one with `rotated_at` null. Older rows stay. HTTP bodies return the fingerprint and expiry.

Pause stores the previous status in `paused_from`. Resume writes that status back.

The master key never encrypts the secret. `Seal` draws a random 32-byte data key, encrypts the secret with AES-256-GCM, and encrypts the data key with the master key. The nonce is prepended. The fingerprint is the hex SHA-256 of the plaintext, so two seals of the same secret match as fingerprints and still use different data keys. `EnvKey` only knows version 1.

`Apply` is the health graph:

- `paused` stays `paused` until resume.
- An empty code means `healthy`.
- `auth_invalid`, `auth_expired`, and `permission_missing` go to `needs_auth` from any other status, including `healthy`.
- `schema_drift` goes to `degraded` when the current status is `healthy`.
- `rate_limited`, `vendor_unavailable`, and `partial_sync` move `healthy` to `degraded`, and move `degraded` or `failing` to `failing`. `needs_auth` stays `needs_auth`.
- Any other code goes to `failing`, except `needs_auth`, which stays.

`Fail` calls `Apply` as if the current status were `healthy`, then writes the result. A paused row is left alone by the update (`status <> 'paused'`). So a failed job does not climb `degraded` to `failing` by itself. `Apply` knows that step. `Fail` does not pass the row's current status in.

## Connectors

A connector has a kind, a description, `Verify`, `Fetch`, and `Normalize`. `Normalize` has to be pure: the same bytes always produce the same rows.

The registry maps kind to connector. `Disabled` is a registered connector that returns an error from `Verify`, `Fetch`, and `Normalize`. The API maps a missing kind to 400 "unknown connector", a disabled connector to 400 "connector is not enabled", and a verify rejection to 401 `auth_invalid`.

`fakevendor` is the connector demos and tests use. Any non-empty secret verifies. A JSON secret is a fault document: `latency` (a Go duration), `status` 429 or 500, `retryAfter` in seconds, `drift` (the fixture's `service` field is renamed `service_v2`), and `restate`. Drift comes back as an error with `SchemaDrift()`. Rate limit and 5xx come back as an error with `Code()` and `GetRetryAfter()`.

`httpclient` retries 429 and 5xx, up to 3 attempts by default, honors `Retry-After`, and adds jitter. `fakevendor` does not need it. A real vendor connector would.

`make new-connector NAME=acme` copies the scaffold with `sed` and refuses to overwrite an existing directory.

Normalized rows are `CostRow` values. The sync runner calls `Normalize` so it can see drift. It discards the rows. There is no `cost_rows` table yet.

## Sync

`sync_jobs.kind` is `scheduled`, `backfill`, or `reconnect`. Status is `queued`, `running`, `succeeded`, `failed`, `cancelled`, or `quarantined`.

Three partial unique indexes matter:

- One `running` job per connection. Backfill chunks for one connection therefore run one after another. Two connections can run at the same time.
- One `scheduled` job in `queued` or `running` per connection.
- One `queued` or `running` job per connection, kind, and window. Enqueue of a duplicate window returns the job already there.

`Claim` takes a due row with `FOR UPDATE SKIP LOCKED`. Due means `queued` and `next_run_at` in the past, or `running` with `lease_until` already past (a worker that died). It then takes `pg_advisory_xact_lock` on `hashtext(connection_id)` and looks again for a different live running job on that connection. If one exists, the claim commits without taking the job. Otherwise it marks the row running, increments `attempt`, and sets `lease_until` to now plus the lease. The runner's lease defaults to 30 seconds. `Heartbeat` extends it, and fails when the row is no longer running.

The scheduler holds `pg_try_advisory_lock` on `hashtext('contract-ops-sync-scheduler')` for the life of its connection. The process that does not get the lock waits for shutdown. The leader calls `EnqueueScheduled` on an interval (default one minute) and unlocks on the way out. That enqueue selects `healthy` connections only. The window is the high watermark, or `now - lookback` when there is no cursor. Lookback defaults to 24 hours. The stream name is `usage`.

A backfill request splits `[start, end)` into UTC days, at most 366, and enqueues each chunk. The handler checks the connection belongs to the caller's workspace.

`Execute` cancels the job if the connection is paused. It loads the active secret, opens it, and fetches the window. Between fetch and commit it checks again for cancellation.

Outcomes:

- Schema drift: insert a quarantined `raw_batches` row, mark the job `quarantined` with `schema_drift`, set the connection to `degraded` unless it is paused, write an activity event `sync.quarantined`. `sync_cursors` is not updated. `Retry` is not called, so the attempt count is not spent on a loop.
- `ErrRejected`: fail immediately with `auth_invalid`.
- A rate-limit or vendor error that exposes `Code()` and `GetRetryAfter()`: requeue. The wait is `time.Second << attempt`, capped at 5 minutes, or `Retry-After` when that is longer. When `attempt` has reached `max_attempts` (default 5), the job fails.
- Anything else from fetch: the same retry path with `vendor_unavailable`.
- Success: one transaction inserts the raw batch (`ON CONFLICT (connection_id, payload_hash) DO NOTHING`), upserts the cursor with `high_watermark = GREATEST(...)`, marks the job succeeded, and sets the connection `healthy` unless it is paused. The activity event is `sync.succeeded`.

`payload_hash` is SHA-256 of the connection id, the window start, the window end, and each payload, separated by zero bytes. A retry of the same page is one row. The next day is a different row.

The worker runs the scheduler plus `WORKER_CONCURRENCY` runner loops (default 2, allowed 1 through 32). It migrates, and it requires `MASTER_KEY`. It builds the same registry as the API: `fakevendor`, plus disabled OpenAI and Anthropic.

Tests cover the claims that are easy to get wrong: three runners and one job insert one batch; drift does not move the watermark and sets `degraded` / `schema_drift`; a three-day backfill never has more than one chunk in flight and ends with three batches; a 429 followed by success writes one batch. The rate-limit test sets `next_run_at` to now before the second claim, because the first backoff is 2 seconds.

## Deploy

The API image is a multi-stage build: `golang:1.26.5`, then distroless static, nonroot. It copies `/api`, `/migrate`, and `/worker`. `ENTRYPOINT` is `/api`. Compose overrides that for the migrate and worker services.

On Railway the API and the web console are separate services. The API build context is the `api` directory. The web build is the Vite app, with `VITE_API_BASE_URL` pointed at the API origin and `CORS_ORIGIN` pointed back at the console. Production Postgres is the Railway Postgres template. The API's `DATABASE_URL` references that service.

A Railway worker is not part of this deploy. The start command on that platform becomes arguments to the entrypoint, and the entrypoint is `/api`, so a second service cannot currently be told to exec `/worker` without changing the image.
