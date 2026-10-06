# Two-minute demo

Use this when walking through the console. The API still owns credentials, sealing, and sync; this script focuses on what reviewers see in the browser.

## Before you present

```bash
make up
cd web && npm run dev
```

Open `http://localhost:3000` and `http://localhost:3000/console`. Paste `BOOTSTRAP_API_KEY` from [`.env.example`](../.env.example) into **Settings** so **Connections**, **Costs**, and backfill work.

Optional: after a backfill, `npm run screenshots` (API + Next dev running) refreshes [`docs/screenshots/`](./screenshots/).

## Walkthrough

1. **Landing** — hero, architecture section (`/#architecture`), **Open console**.
2. **Activity** — live indicator, fixture table; generator adds events every ~5s up to eight total.
3. **Filters** — search `fireblocks`, status **Processed**; URL carries `q` and `status`. Search `zzznomatch` for the empty state.
4. **Settings** — workspace API key and API base URL (local: `http://localhost:8080`).
5. **Connections** — two seeded `fakevendor` rows after a fresh volume; **Verify** and **Backfill 24h** (worker must run locally via `make up`).
6. **Costs** — after backfill completes, normalized billing rows from `GET /v1/costs`.
7. **Depth** (if asked) — `openapi.yaml`, sync engine notes in [`rehearse-sync-engine.md`](./rehearse-sync-engine.md), `docs/architecture.md`.

## Verify before merge

```bash
npm run check
cd web && npm run lint && npm run typecheck && npm run build
cd api && go test ./...
npm run test:e2e    # Postgres on :5432; API :18080 + web :13000 (see docs/e2e.md)
```

The demo generator stops at eight fixtures; it is not vendor sync.
