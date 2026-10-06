# Architecture Notes

## Why this shape

Contract Ops is a portfolio slice of the kind of work web teams do when a React/TypeScript front end talks to a small back-end service:

- the UI stays focused on interaction and presentation
- the API owns validation, filtering, and response shape
- the OpenAPI file is the shared contract
- tests cover the behaviour that is easy to regress

## Boundaries

`internal/events` owns fixture data, live append generation, sorting, filtering, and the in-memory feed used by tests. `internal/activity` stores the same feed in Postgres for the running API.

`internal/httpapi` owns HTTP parsing, status validation, CORS, JSON list responses, and the SSE stream.

`cmd/api` only composes those pieces, starts the generator, and shuts down on signal.

Frontend:

- `src/api` builds stream URLs and validates payloads at runtime
- `src/domain` holds shared types
- `src/app` owns the EventSource lifecycle
- `src/features/events` renders controls and results

## Request flow

```text
Search or status selection
        |
        v
React opens EventSource
        |
        v
GET /v1/events/stream?q=...&status=...
        |
        v
Go handler validates status
        |
        v
SSE sends filtered EventPage snapshot
        |
        +---- generator appends fixture every 5s (until 8)
        |
        v
SSE pushes updated EventPage
        |
        v
Runtime validation → loading / results / empty / error
```

`GET /v1/events` remains available as a plain JSON list for contract inspection and non-stream clients.

## Maintainability and reliability choices

In a larger system these same ideas scale up:

- keep transport details out of presentational components
- validate at the HTTP boundary before touching domain logic
- make empty, loading, and failure states first-class
- prefer deterministic fixtures for local demos and tests
- shut down cleanly so in-flight work does not leave the process hanging
- document the contract so front end and back end can evolve independently

This repository keeps those habits visible without pretending to be a multi-service production platform.

## Accessibility choices

Native labelled controls, visible focus, polite live-region summaries, alert roles for errors, and status that is not colour-only. Loading appears when filters change, not on every SSE tick, so live updates do not thrash the status text. The table scrolls horizontally on narrow viewports instead of collapsing into ambiguous cards.

## Screenshots

See [`screenshots/`](./screenshots/) and the gallery in the root README.
