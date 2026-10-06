# Contract Ops

Contract Ops is an integrations control plane with an event console in front of it. The console shows a live activity feed. The API stores connections, seals vendor credentials, and queues sync jobs. A worker claims those jobs.

The **web app** is Next.js 15 and TypeScript (landing page, animated architecture, and `/console`). A legacy Vite bundle remains at the repo root for local tooling. The API, worker, and migrations are Go. Postgres holds the data. The HTTP contract is OpenAPI 3.1 in [`openapi.yaml`](./openapi.yaml).

## What is running

`api` serves HTTP. It migrates on boot, then listens. `worker` claims sync jobs. Compose starts one. On Railway the API, Next console, and a separate `worker` service share the same Postgres (`docs/deploy-railway.md`). `migrate` is the same goose binary Compose and CI call directly. The image also contains it, but the container entrypoint is `/api`, so the API and the worker apply migrations themselves before they serve.

The console reads the **public** activity feed over SSE. With a workspace API key (Settings), it also calls connection and sync routes over ordinary `fetch` requests.

### Live

| Service | URL |
|---------|-----|
| API | https://api-production-4b82.up.railway.app |
| Console | https://web-production-a614a.up.railway.app |

Railway service roots and worker setup: [`docs/deploy-railway.md`](docs/deploy-railway.md).

### Console (at a glance)

![Landing page](docs/screenshots/01-landing.png)

- **Activity** — live feed, search, status and connection filters, shareable URLs, load older pages via cursor.
- **Connections** — list, create (`fakevendor` demo), verify, queue a 24-hour backfill (requires API key).
- **Costs** — workspace billing rows from `GET /v1/costs` (requires API key; needs worker + backfill for data).
- **Settings** — API base URL and bearer key (stored in the browser only).

Screenshots (activity, connections, costs, settings, empty state): [`docs/screenshots/`](docs/screenshots/). Regenerate with `make up`, `cd web && npm run dev`, run a connection backfill so **Costs** has rows, then `npm run screenshots` from the repo root.

`/healthz` means the process is up. `/readyz` means Postgres answers and its goose version matches the migrations compiled into that binary.

## Read next

- [Architecture](docs/architecture.md) — how a request, a credential, and a sync job move.
- [Codebase](docs/codebase.md) — what each file is doing.
- [Demo script](docs/demo-script.md) — the console walkthrough.
- [Product improvements](docs/product-improvements.md) — roadmap and research notes.

## Run locally

Node.js 18 or newer, npm, Go 1.26 or newer, and Docker.

```bash
make up
cd web && npm install && npm run dev
```

`make up` starts Postgres 16, applies migrations, starts the API on port 8080, and starts a worker. Open `http://localhost:3000` for the marketing site and `http://localhost:3000/console` for the control plane UI. (Legacy Vite dev server: `npm run dev` at repo root on port 5173.) The feed begins with four fixtures. On first boot the API also seeds two verified `fakevendor` connections for workspace `ws_local` when that workspace has none. The demo generator appends one about every five seconds until eight events are stored. Those rows stay in the Postgres volume, so the next `make up` does not grow the list again. `docker compose down -v` drops the volume.

`DEMO_GENERATOR` defaults to true. Set it to false when you want the fixtures and nothing further.

To run the API on the host, point it at a database Compose has already migrated:

```bash
cd api
DATABASE_URL=postgres://contract_ops:contract_ops@localhost:5432/contract_ops?sslmode=disable go run ./cmd/api
```

The worker is `go run ./cmd/worker`. `WORKER_CONCURRENCY` defaults to 2 and must be from 1 to 32. Both processes require `MASTER_KEY`. The value in [`.env.example`](./.env.example) is a local development key, the same kind of secret as the local database password.

## Verify

```bash
make test
make lint
```

`make test` runs `go test ./...` and `npm run check`. Integration tests skip unless `TEST_DATABASE_URL` is set. They create and drop their own databases. They do not truncate the Compose demo database. CI sets that URL.

Playwright E2E (`npm run test:e2e`) needs Postgres on localhost; see [`docs/e2e.md`](docs/e2e.md). CI runs the same suite in the `e2e` job.

`make new-connector NAME=acme` copies `api/internal/connectors/scaffold` into a new package. The name has to be a lowercase identifier, and the directory must not already exist.

## API, short version

These routes are public. The browser's `EventSource` cannot set `Authorization`, so the feed stays open.

| Method and path | What it returns |
|---|---|
| `GET /healthz` | `{"status":"ok"}` |
| `GET /readyz` | the same, or a problem response when the schema does not match |
| `GET /metrics` | Prometheus text |
| `GET /v1/events` | one activity page |
| `GET /v1/events/stream` | the same page, pushed as SSE when the table changes |

`q` is a case-insensitive match on id, source, type, or correlation id. `status` is `processed`, `pending`, `failed`, or `all`. `limit` defaults to 50 and stops at 100. `cursor` is the opaque keyset from the previous page. `total` counts the whole filtered set, not the page.

Every other route expects `Authorization: Bearer`. On boot, `BOOTSTRAP_API_KEY` is hashed and stored for workspace `ws_local`. The secret is not stored. The local key is in `.env.example`.

`POST`, `PUT`, and `PATCH` accept an optional `Idempotency-Key` once a workspace is on the request. The same key and the same body replay the stored response. A different body, or a request still in progress, returns 409.

Connections: create, list, get, patch name or pause, verify, and rotate. Kinds are `fakevendor`, `openai`, and `anthropic`. A new connection starts at `needs_auth`. Verify moves a good secret to `healthy`. Rotate checks the new secret before it replaces the stored one. Responses carry a fingerprint, never the secret. `fakevendor` accepts any non-empty secret, or a JSON fault document. OpenAI and Anthropic verify API keys against the vendor and ingest organization cost payloads on sync.

`GET /v1/costs` lists normalized billing rows for the workspace (optional `connection_id`, `start`, `end`, cursor pagination). Rows are written when a sync job commits a page.

`POST /v1/connections/{id}/backfill` takes `start` and `end` and returns 202 with one job per UTC day, from 1 to 366 days. `POST /v1/sync/jobs/{id}/cancel` cancels a queued or running job in that workspace.

Sync engine walkthrough for interviews: [`docs/rehearse-sync-engine.md`](docs/rehearse-sync-engine.md).

Failures use `application/problem+json`. The code list is [`api/openapi/errors.yaml`](./api/openapi/errors.yaml).

## Licence

MIT. See [`LICENSE`](./LICENSE).
