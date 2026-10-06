# Railway (production)

Monorepo layout: **`api/`** (Go API + worker binaries in one image), **`web/`** (Next.js console).

## Service roots

| Service | Root directory | Build | Start |
|---------|----------------|-------|-------|
| `api` | `api` | `api/Dockerfile` | `/api` (image entrypoint) |
| `web` | `web` | `web/Dockerfile` | `node server.js` |
| `worker` (optional) | `api` | same image as API | `/worker` |

Set **Root Directory** on each service in Railway (or `serviceInstanceUpdate` with `rootDirectory: "api"` / `"web"`). If the API service builds from the repo root, Railpack will run the root `package.json` and `/healthz` will 404.

API env: `DATABASE_URL`, `MASTER_KEY`, `BOOTSTRAP_API_KEY`, `CORS_ORIGIN` (console origin), `DEMO_GENERATOR`.

Web env: `NEXT_PUBLIC_API_BASE_URL` → API public URL.

Worker uses the same `DATABASE_URL` and `MASTER_KEY` as the API; no HTTP port.

## Deploy

```bash
git push origin main   # GitHub-connected services redeploy
# or from service roots:
cd api && railway up --detach
cd web && railway up --detach
```

Health: `GET /healthz` and `GET /readyz` on the API URL.
