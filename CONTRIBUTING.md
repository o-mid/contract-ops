# Contributing

Keep a change small enough to review in one sitting, and keep the HTTP contract easy to point at.

## Before opening a change

1. Read `README.md`, `docs/architecture.md`, and `openapi.yaml`.
2. Put feed behaviour in `api/internal/events` and `api/internal/activity`. Put HTTP parsing in `api/internal/httpapi`. Connection rules go in `api/internal/connections`. Vendor calls go in `api/internal/connectors`. Job claiming goes in `api/internal/sync`.
3. Keep console components on rendering. Stream URLs and payload checks stay in `src/api`. The `EventSource` lifecycle stays in `src/app`.
4. Add or update a focused test for behaviour a reviewer could regress. Postgres tests create their own database and skip when `TEST_DATABASE_URL` is unset. Do not point them at the Compose demo database.
5. If the console's behaviour changes, update the screenshots under `docs/screenshots/` when you can.
6. A new error code goes in `api/openapi/errors.yaml`, then `go run ./cmd/gencatalog` from `api/`. A new request or response field goes in `openapi.yaml`, then the model generator. CI rejects a drift in either generated file.
7. A new migration needs a matching assertion in `api/internal/platform/ready/ready_test.go`, which checks the latest embedded version.

```bash
make test
make lint
```

## Accessibility baseline

- Use native controls before adding ARIA.
- Every form control needs a visible label.
- Keyboard focus must remain visible.
- Status changes, failures, and empty states must be understandable without colour alone.
- Test a narrow layout as well as a desktop viewport.

## Commit shape

Subject in the imperative, lowercase, 72 characters or fewer. A short body when the reason is not obvious from the subject.

- `feat(sync): lease jobs and keep backfill serial`
- `fix(connections): verify a secret before swapping it`
- `docs: describe the control plane as it runs`
