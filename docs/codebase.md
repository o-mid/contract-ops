# Codebase

This is a map of the tree as it is, one file at a time. Generated files say where they come from. Tests say what they lock in.

## Repository root

`README.md` is the way in: how to run the stack, which routes are public, and where the longer notes live.

`CONTRIBUTING.md` is the short review checklist: where a change belongs, the accessibility bar for the console, and the commands to run.

`LICENSE` is the MIT licence.

`openapi.yaml` is the HTTP contract. It covers health, ready, metrics, the event feed, connections, credential rotate, backfill, and cancel. Handlers are written by hand against this file. Models are generated from it. The description at the top is about the whole API, not only the feed.

`compose.yaml` runs Postgres 16, a one-shot migrate, the API on port 8080, and a worker with `WORKER_CONCURRENCY` 2. The worker publishes no ports. The database password and the example master key in this file are local defaults.

`.env.example` is the same local settings for a process you start on the host. `DATABASE_URL` there uses `localhost`. Compose rewrites the host to `postgres` for the containers.

`Makefile` runs `make test` (Go tests, then `npm run check`), `make lint` (golangci-lint v2.13.2, installed into `bin/`), `make up`, `make down`, and `make new-connector`.

`.github/workflows/ci.yml` has two jobs. `web` is Node 20, `npm ci`, `npm run check`. `api` is Postgres 16, then `gencatalog -check`, an `oapi-codegen` regenerate with `git diff --exit-code`, goose `up`, `down`, `down`, `up`, `go test`, and golangci-lint. `DATABASE_URL` and `TEST_DATABASE_URL` both point at the CI database.

`package.json` holds the console scripts. `npm run check` is eslint, then vitest, then `tsc -b` and the Vite build. `npm run screenshots` drives Playwright against a running dev server. Runtime dependencies include React 18.3, TanStack Query, Lucide icons, and Tailwind (via PostCSS). Node 18 is the floor this repo declares. CI uses 20.

`tailwind.config.js` and `postcss.config.js` configure the console stylesheet. `scripts/capture-screenshots.mjs` writes PNGs under `docs/screenshots/`.

`vite.config.ts` is the Vite and Vitest config: the React plugin, jsdom, and `src/test/setup.ts`.

`index.html` is the shell Vite serves. The console mounts on `#root`.

`src/styles.css` pulls in Tailwind layers and a few global resets.

## Console

`src/main.tsx` mounts `App` inside `QueryClientProvider` and `SettingsProvider`.

`src/vite-env.d.ts` types `import.meta.env`, including `VITE_API_BASE_URL`.

`src/test/setup.ts` loads the jest-dom matchers for Vitest.

`src/lib/cn.ts` merges Tailwind class names. `src/lib/format.ts` formats UTC timestamps.

`src/domain/event.ts` is the feed shape, filters (including optional `connectionId`), and loading / ready / error state. `nextCursor` is used for “load older” via `GET /v1/events`.

`src/domain/connection.ts` types connections and sync jobs returned by the API.

`src/domain/errorCatalog.gen.ts` is generated from `api/openapi/errors.yaml`. `src/api/client.ts` maps problem `code` values through the catalog for user-facing errors.

`src/api/events.ts` builds stream and page URLs from the configured API base. `parseEventPage` validates payloads at runtime.

`src/api/connections.ts` lists, creates, verifies, and backfills connections over authenticated `fetch`.

`src/app/settings.tsx` and `src/app/settingsStorage.ts` persist API base URL and bearer key in `localStorage`. `src/app/useSettings.ts` reads them.

`src/app/useUrlFilters.ts` syncs view and feed filters to the query string.

`src/app/App.tsx` wires navigation, deferred filters, and a single `useEvents` instance for the shell live indicator.

`src/app/useEvents.ts` owns the `EventSource` lifecycle, optional cursor pagination, and retry.

`src/app/App.test.tsx` mocks `EventSource` for the activity view. Five tests.

`src/components/layout/AppShell.tsx` is the sidebar shell. `src/components/ui/` holds buttons and badges.

`src/pages/EventsPage.tsx`, `ConnectionsPage.tsx`, and `SettingsPage.tsx` are the three main views.

`src/features/events/` contains the toolbar, table, and status badges.

`docs/screenshots/` holds PNGs of the console states. `docs/product-improvements.md` records roadmap notes. `docs/demo-script.md` is the spoken walkthrough.

## API process

`api/go.mod` is the Go module `github.com/o-mid/contract-ops/api`.

`api/Dockerfile` builds `/api`, `/migrate`, and `/worker` with `CGO_ENABLED=0`, then copies them onto distroless static, user `nonroot`. `ENTRYPOINT` is `/api`. Compose replaces the entrypoint for migrate and the worker. A host that only sets a start command does not.

`api/tools.go` is built with the `tools` tag so `oapi-codegen` stays in `go.mod` without being linked into the server.

`api/oapi-codegen.yaml` tells the generator to emit models, not handlers, into `internal/platform/apigen`.

`api/.golangci.yml` is golangci-lint v2: errcheck, govet, ineffassign, staticcheck, unused, and gofmt.

`api/cmd/api/main.go` wires the process. Order: load config, migrate, open the pool, seed the bootstrap key, start `LISTEN`, insert fixtures, build the sealer and the connector registry, mount connection and sync routes, start the demo generator if it is enabled, serve, shut down on signal. `registryVerifier` at the bottom of the file is the production `Verifier`. A missing kind and a disabled connector are 400. A verify error is 401 `auth_invalid`.

`api/cmd/worker/main.go` is the same migrate-then-pool boot, without HTTP. It requires the master key, builds the same registry, and runs one scheduler plus N runners until a signal or a real error. `applyMigrations` is duplicated here rather than shared with `cmd/api`.

`api/cmd/migrate/main.go` reads `DATABASE_URL` and applies the embedded migrations. The argument is `up` or `down`. `up` is the default.

`api/cmd/gencatalog/main.go` reads `openapi/errors.yaml` and writes the Go catalog and the TypeScript catalog. `-check` exits non-zero when either file would change.

## Migrations

`api/migrations/embed.go` embeds every `*.sql` file so the binary does not need the source tree mounted.

`api/migrations/00001_activity_events.sql` creates the feed, the newest-first keyset index, and a partial index on `connection_id` for rows that have one. Demo rows leave that column null.

`api/migrations/00002_activity_notify.sql` adds the statement-level trigger. The payload is `''`.

`api/migrations/00003_workspaces.sql` creates `workspaces`, `api_keys` (hash, prefix, unique among live hashes), and `idempotency_keys`. Status 0 on an idempotency row means the request has claimed the key and has not finished.

`api/migrations/00004_connections.sql` creates `connections` and `credentials`. One active credential per connection: `rotated_at IS NULL`.

`api/migrations/00005_sync.sql` creates `sync_jobs`, `sync_cursors`, and `raw_batches`, including the unique indexes that keep one running job per connection and one active scheduled job per connection. `payload_hash` is unique per connection. The comments in that file are the constraint list. Read them with `internal/sync`.

## Platform

`api/internal/platform/config/config.go` parses the environment and fails closed. `config_test.go` covers defaults, overrides, a missing or non-postgres `DATABASE_URL`, rejected values such as port 0, and a bootstrap key that is too short.

`api/internal/platform/log/log.go` is a JSON `slog` handler on stdout.

`api/internal/platform/db/db.go` opens the pgx pool with the limits described in the architecture notes. `db_test.go` checks that an unparseable URL is an error.

`api/internal/platform/migrate/migrate.go` sets up goose against the embedded filesystem and exposes `Up` and `Down`.

`api/internal/platform/ready/ready.go` is the `/readyz` check. `ready_test.go` asserts the latest embedded version. Today that number is 5.

`api/internal/platform/httpx/middleware.go` is request id, CORS, access log, recover, body limit, and timeout. The timeout wrapper's `skip` function is how the event stream stays open. `middleware_test.go` covers request ids, recovery, CORS, and that skip.

`api/internal/platform/problem/problem.go` writes the problem+json body and fills catalog fields when the code is known. `problem_test.go` checks a known code and an unknown one.

`api/internal/platform/catalog/catalog.gen.go` is generated. Do not edit it.

`api/internal/platform/apigen/types.gen.go` is generated from `openapi.yaml`. Do not edit it.

`api/internal/platform/telemetry/metrics.go` is the private registry and the middleware. The status recorder forwards `Flush`.

`api/internal/platform/auth/auth.go` hashes keys, seeds `ws_local`, and skips the public paths listed in the architecture notes. `WithWorkspace` is exported so tests can put a workspace on a context. `auth_test.go` checks that the event feed stays open and that another route requires a known key. `store_test.go` is the Postgres test for bootstrap and resolve. It creates `contract_ops_auth_test` and drops it. It skips when `TEST_DATABASE_URL` is unset.

`api/internal/platform/idempotency/idempotency.go` is claim, complete, release, and the middleware that buffers the response. `idempotency_test.go` and `store_test.go` use `contract_ops_idem_test`. Same key and body replays. A different body conflicts. An in-progress claim conflicts.

## Events and the feed

`api/internal/events/store.go` is the domain type and the in-memory `Store`: filter, keyset page, subscribe. `MaxEvents` is 8. Default page size is 50, max 100.

`api/internal/events/cursor.go` encodes and decodes the keyset token. A token this process cannot read is `ErrInvalidCursor`.

`api/internal/events/fixtures.go` is the four starting events and the four the generator may append.

`api/internal/events/generator.go` appends those pending fixtures on a ticker until the feed reaches `MaxEvents` or the context is cancelled. An id that is already stored does not stop the walk early.

`api/internal/events/store_test.go` and `generator_test.go` cover filtering, the cursor, and the cap. They do not need Postgres.

`api/internal/activity/store.go` is the Postgres `Feed`. `Append` notifies in-process subscribers. `Listen` subscribes to `activity_events` and returns a function that stops that loop. Callers defer it so it runs before the pool closes.

`api/internal/activity/listen_test.go` uses database `contract_ops_test`. It checks that a second store, listening, wakes up when the first store inserts.

`api/internal/httpapi/server.go` mounts middleware and the event routes, and calls `Options.Register` for everything else. `listEvents` and `streamEvents` validate `status` before they read the feed. The stream writes a snapshot, then waits on the subscription. `server_test.go` uses the memory store for ready, a filtered list, a bad status, and a stream snapshot.

## Connections

`api/internal/connections/model.go` is the connection and credential types, and the status constants.

`api/internal/connections/health.go` is `Apply`, the graph in the architecture notes. `health_test.go` walks the edges, including a jump from `healthy` to `needs_auth` and a paused row that does not move.

`api/internal/connections/store.go` is the SQL: create, list, get, pause and resume, rotate, and the active secret. List and get are scoped by workspace.

`api/internal/connections/service.go` seals on create, verifies on verify, and verifies before swap on rotate. `StaticVerifier` accepts any non-empty `fakevendor` secret and rejects the other kinds. Production does not use it. `cmd/api` passes `registryVerifier`. Tests do.

`api/internal/connections/handler.go` is the HTTP surface: `POST/GET /v1/connections`, `GET/PATCH /v1/connections/{id}`, `POST .../verify`, `POST .../rotate`. Patch applies name and pause when either is set. Bodies are `CreateInput` and `RotateInput`, not a second struct of the same shape.

`api/internal/connections/http_test.go` uses `contract_ops_conn_test`. One test walks create, verify, a rejected rotate, a successful rotate, and pause then resume. The secret never appears in a JSON body, and a rejected rotate leaves the previous secret in place.

`api/internal/credentials/seal.go` is the envelope: random data key, AES-256-GCM, master key wraps only that data key. `seal_test.go` checks a round trip where the plaintext is absent from the ciphertext, and that two seals of one plaintext use different wrapped keys.

## Connectors

`api/internal/connectors/connector.go` is the interface, `CostRow`, and `ErrRejected`.

`api/internal/connectors/registry.go` is the kind map and `Disabled`. `registry_test.go` checks lookup and a disabled verify.

`api/internal/connectors/httpclient/client.go` retries 429 and 5xx. `client_test.go` checks `Retry-After` and that a 200 stops the loop.

`api/internal/connectors/fakevendor/fakevendor.go` embeds `testdata/record.json` and implements the fault document. `fakevendor_test.go` compares `Normalize` with `testdata/golden.json`, and checks drift, 429, and 500.

`api/internal/connectors/scaffold/scaffold.go` is the copy `make new-connector` starts from. `scaffold_test.go` is the matching test, also rewritten by that sed. The scaffold is not registered in `cmd/api`.

## Sync

`api/internal/sync/job.go` is the job and batch structs and the `usage` stream name.

`api/internal/sync/chunks.go` splits a backfill into UTC days. `backoff` starts at one second shifted by the attempt and caps at five minutes. `chunks_test.go` checks the day boundaries and the cap.

`api/internal/sync/store.go` is enqueue, claim, heartbeat, cancel, commit, quarantine, retry, and fail. `Claim` is the `SKIP LOCKED` query plus the per-connection advisory lock. `Quarantine` does not write `sync_cursors`. `CommitPage` writes the batch and the cursor in one transaction.

`api/internal/sync/runner.go` is `Execute`: pause cancels, drift quarantines, a rejected credential fails, a rate limit retries, success commits. `hashPage` is the raw-batch identity.

`api/internal/sync/scheduler.go` is the leader lock and the enqueue interval.

`api/internal/sync/handler.go` is backfill and cancel. Backfill refuses a range that is empty or longer than 366 days.

`api/internal/sync/runner_test.go` is the Postgres proof, database `contract_ops_sync_test`: one batch under three runners, drift leaves the watermark, a three-day backfill stays serial, and a 429 then a success is one batch.

## What is not in this tree

Normalized cost rows are not stored. OpenAI and Anthropic have no vendor calls. The console has no connection or sync screens. Railway is not running a worker. Those are the edges of the current code, not stubs hiding in a handler that returns 501.
