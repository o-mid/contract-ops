# Contributing

This repository is a skills showcase. Keep changes small enough to review in one sitting, and keep the front-end / API contract easy to explain.

## Before opening a change

1. Read `README.md` and `openapi.yaml`.
2. Put API filtering and response-shape logic in `api/internal`.
3. Keep UI components focused on rendering and interaction. Keep stream/fetch behaviour in `src/api` or `src/app`.
4. Add or update a focused test for changed behaviour.
5. If UI behaviour changes, update screenshots under `docs/screenshots/` when practical.
6. Run the checks below.

```bash
npm run check
cd api && go test ./...
```

## Accessibility baseline

- Use native controls before adding ARIA.
- Every form control needs a visible label.
- Keyboard focus must remain visible.
- Status changes, failures, and empty states must be understandable without colour alone.
- Test narrow layouts as well as a desktop viewport.

## Commit shape

Use one concern per commit. Good examples:

- `add event filtering contract`
- `render accessible event results`
- `test invalid event status`
