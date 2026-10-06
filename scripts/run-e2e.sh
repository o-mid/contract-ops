#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export DATABASE_URL="${DATABASE_URL:-postgres://contract_ops:contract_ops@localhost:5432/contract_ops?sslmode=disable}"
export BOOTSTRAP_API_KEY="${BOOTSTRAP_API_KEY:-co_local_dev_key_not_for_production}"
export MASTER_KEY="${MASTER_KEY:-MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=}"
export CORS_ORIGIN="${CORS_ORIGIN:-http://localhost:13000}"
export DEMO_GENERATOR="${DEMO_GENERATOR:-true}"
# Dedicated ports so e2e does not fight `make up` on 8080/3000.
export PORT="${PORT:-18080}"
export E2E_API_BASE="${E2E_API_BASE:-http://localhost:${PORT}}"
export PLAYWRIGHT_BASE_URL="${PLAYWRIGHT_BASE_URL:-http://localhost:13000}"
export WEB_PORT="${WEB_PORT:-13000}"

cleanup() {
  [[ -n "${API_PID:-}" ]] && kill "$API_PID" 2>/dev/null || true
  [[ -n "${WORKER_PID:-}" ]] && kill "$WORKER_PID" 2>/dev/null || true
  [[ -n "${WEB_PID:-}" ]] && kill "$WEB_PID" 2>/dev/null || true
}
trap cleanup EXIT

echo "migrate…"
(cd "$ROOT/api" && go run ./cmd/migrate)

echo "api…"
(cd "$ROOT/api" && go run ./cmd/api) &
API_PID=$!

echo "worker…"
(cd "$ROOT/api" && go run ./cmd/worker) &
WORKER_PID=$!

for _ in $(seq 1 90); do
  if curl -sf "$E2E_API_BASE/healthz" >/dev/null; then
    break
  fi
  sleep 1
done
curl -sf "$E2E_API_BASE/readyz" >/dev/null

echo "web build + start…"
export NEXT_PUBLIC_API_BASE_URL="$E2E_API_BASE"
(cd "$ROOT/web" && npm run build)
(cd "$ROOT/web" && PORT="$WEB_PORT" npm run start) &
WEB_PID=$!

for _ in $(seq 1 90); do
  if curl -sf "$PLAYWRIGHT_BASE_URL/console" >/dev/null; then
    break
  fi
  sleep 1
done

cd "$ROOT"
npx playwright test "$@"
