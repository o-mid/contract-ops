# Contract Ops

Contract Ops is a small, typed service-event browser. It is deliberately narrow: a React and TypeScript interface consumes a documented Go REST API with deterministic fixtures.

The project exists to make a few engineering decisions easy to review:

- semantic controls and keyboard-operable filtering
- responsive CSS that keeps the table usable on narrow screens
- explicit loading, empty, and error states
- a small HTTP boundary with validation and deterministic data
- focused frontend and Go tests

## Architecture

```text
src/
  api/              HTTP client and response boundary
  domain/           Shared frontend types
  features/         Event filter and table UI
  app/              Page composition and request lifecycle
api/
  cmd/server/       Process entry point
  internal/events/  Event model, fixtures, filtering
  internal/httpapi/ HTTP routing and response boundary
openapi.yaml        Public API contract
```

The frontend owns UI state and sends filters as query parameters. The API owns filtering and response shape. Fixtures are fixed so a fresh clone has predictable behaviour.

## Run locally

Requirements: Node.js 18+, npm, and Go 1.26+.

```bash
npm install
cd api && go test ./... && go run ./cmd/server
```

In another terminal:

```bash
npm run dev
```

Open the Vite URL, normally `http://localhost:5173`.

## Verify

```bash
npm run check
cd api && go test ./...
```

## API

`GET /v1/events` accepts:

| Parameter | Meaning |
|---|---|
| `q` | Case-insensitive match against event ID, source, type, or correlation ID |
| `status` | `processed`, `pending`, `failed`, or `all` |

The complete contract is in [`openapi.yaml`](./openapi.yaml).

For a short walkthrough, use the [demo script](./docs/demo-script.md).

## Scope

There is no database, authentication, pagination, or event broker. Those are intentional omissions. The point is a clean and testable contract between an accessible UI and a small service, not a simulated production platform.

## Licence

This project is released under the [MIT Licence](./LICENSE).
