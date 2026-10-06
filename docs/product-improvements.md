# Product and engineering improvements

This note captures a pass over Contract Ops in early 2026: what the repo already does well, what we shipped in the console redesign, and what is still worth doing.

## What the repo is

Contract Ops is an **integrations control plane** with a **live activity console**. The Go API stores workspace connections, seals vendor credentials, and queues sync jobs. A worker claims those jobs and talks to connectors. Postgres holds state; the React console reads the **public** event feed over server-sent events (SSE).

Live deployments:

| Surface | URL |
|--------|-----|
| API | https://api-production-4b82.up.railway.app |
| Console | https://web-production-a614a.up.railway.app |

Local: `make up`, then `npm run dev` → http://localhost:5173 (API on :8080).

## Strengths today

- Clear split: public feed vs bearer-protected control routes.
- Postgres `LISTEN/NOTIFY` keeps SSE snapshots honest without pushing stale payloads.
- OpenAPI 3.1 contract, generated error catalog, hand-written handlers.
- Demo generator makes the console easy to show without a real vendor.

## Shipped in the console redesign

- **Shell navigation**: Activity, Connections, Settings (dark ops layout, Tailwind).
- **URL-synced filters** on the feed (`q`, `status`, `connection`, `view`).
- **Settings**: API base URL and workspace key (localStorage); unlocks authenticated routes.
- **Connections**: list, create (`fakevendor`), verify, 24h backfill; problem+json surfaced via the generated catalog.
- **Feed**: connection filter when keys + connections exist; optional `connectionId` on rows; **Load older events** via cursor pagination on `GET /v1/events`.
- **Stack**: TanStack Query for REST, Lucide icons, shared API client.

Reference patterns we aligned with (not copied verbatim): [Nango webapp](https://github.com/NangoHQ/nango) (Vite + table/query split), [Windmill](https://github.com/windmill-labs/windmill) (SSE constraints), [Hookdeck Outpost](https://github.com/hookdeck/outpost) (event destination vocabulary).

## Shipped: billing core (P6–P7)

- `cost_rows` table; runner persists normalized rows on successful commit (same transaction as `raw_batches`).
- `GET /v1/costs` with workspace scoping, connection filter, and keyset pagination.
- **OpenAI** and **Anthropic** connectors: verify via models API, fetch organization cost payloads, normalize to FOCUS-shaped rows.
- Rehearsal notes: [`rehearse-sync-engine.md`](./rehearse-sync-engine.md).

## Recommended next (backend + console)

| Priority | Item | Why |
|----------|------|-----|
| High | `GET /v1/sync/jobs` (list/filter) | Backfill returns job IDs only; ops need a queue view. |
| High | Railway worker service | README notes API + web only; sync jobs need a worker in prod. |
| Medium | Wire `errorCatalog` in all console error states | Catalog exists; connections page uses it first. |
| Medium | E2E smoke (Compose + Playwright) | CI has no browser path against a running API. |
| Medium | OpenAPI: document feed 400s for bad cursor/limit | Handlers already return them; spec lags. |
| Lower | Refine or full admin framework | Only if CRUD surface grows past 3–4 resources. |
| Lower | Tokenized SSE URL | If the feed must be authenticated without query tokens. |

## Testing gaps to close

- HTTP tests for sync backfill/cancel handlers.
- Frontend tests for `parseEventPage`, connections API parsers, Settings flow.
- Activity store integration tests against Postgres (filters + cursor).

## Accessibility

The redesign keeps native controls, visible labels, focus rings, and `aria-live` on result counts. Re-check keyboard order on Connections forms and the mobile section switcher when you change layout.
