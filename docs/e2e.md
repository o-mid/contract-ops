# End-to-end tests

Playwright covers the marketing site, public console feed, authenticated console routes, and the billing HTTP API.

## Run locally

Requires Postgres on `localhost:5432` (e.g. `make up` or Compose), Go 1.26+, and Node 20.

```bash
npm ci
npm ci --prefix web
npm run test:e2e
```

`scripts/run-e2e.sh` migrates, starts the API and worker, builds and serves the Next console, then runs `playwright test`. Default ports are **18080** (API) and **13000** (web) so the suite can run alongside `make up` on 8080/3000.

Environment overrides: `PORT`, `WEB_PORT`, `E2E_API_BASE`, `E2E_API_KEY`, `PLAYWRIGHT_BASE_URL`, `DATABASE_URL`.

## CI

The `e2e` job in [`.github/workflows/ci.yml`](../.github/workflows/ci.yml) runs the same script against a service Postgres container.

## Specs

| File | Coverage |
|------|----------|
| `e2e/landing.spec.ts` | Home, architecture anchor, console links |
| `e2e/console-activity.spec.ts` | Public activity feed, URL filters, empty state |
| `e2e/console-auth.spec.ts` | Settings key, seeded connections, costs after backfill |
| `e2e/api.spec.ts` | `/healthz`, `/readyz`, `/v1/costs` auth and ingest |
