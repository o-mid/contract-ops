# Architecture Notes

## Deliberate boundaries

The API is intentionally small. `internal/events` owns fixture data, sorting, and filtering. `internal/httpapi` owns HTTP parsing, status validation, CORS, and JSON responses. The server entry point only composes those pieces.

The frontend keeps the same separation:

- `src/api` converts an HTTP response into a typed page.
- `src/domain` holds the event and filter types.
- `src/app` manages the request lifecycle.
- `src/features/events` renders controls and results.

This avoids putting fetch logic in table components or leaking transport details across the UI.

## Request flow

```text
Search or status selection
        |
        v
React request state
        |
        v
GET /v1/events?q=...&status=...
        |
        v
Go handler validates status
        |
        v
Fixture store filters and orders results
        |
        v
Typed JSON page rendered as loading, results, empty, or error
```

## Tradeoffs

There is no persistence because deterministic fixture data makes the contract, tests, and local demo repeatable. There is no pagination because the fixture set is small. A production version would add a storage boundary, pagination, authentication, rate limiting, tracing, and more specific CORS policy before it added more screen features.

## Accessibility choices

The filter uses native labelled controls. Result updates are announced through a live region. Errors use an alert, status is not colour-only, and visible focus styles are kept. The table is horizontally scrollable on narrow viewports rather than collapsing semantic columns into an ambiguous card layout.
