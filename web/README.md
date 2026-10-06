# Contract Ops web

Next.js 15 app: marketing landing (`/`), animated architecture section, and the operations console (`/console`).

## Local

```bash
make up   # API on :8080
npm ci
npm run dev
```

Open http://localhost:3000 and http://localhost:3000/console. In **Settings**, paste `BOOTSTRAP_API_KEY` from the repo root `.env.example`.

`NEXT_PUBLIC_API_BASE_URL` defaults to `http://localhost:8080`.

## Railway

This folder is the deploy root for the **web** service.

1. Service root directory: `web`
2. Builder: Dockerfile (`web/Dockerfile`)
3. Variables:
   - `NEXT_PUBLIC_API_BASE_URL` = your API URL (e.g. `https://api-production-4b82.up.railway.app`)
4. On the **api** service, set `CORS_ORIGIN` to the console origin (e.g. `https://web-production-a614a.up.railway.app`) so browser `fetch` from Settings/Connections works.

```bash
cd web
railway link   # project contract-ops, service web
railway variables set NEXT_PUBLIC_API_BASE_URL=https://api-production-4b82.up.railway.app
railway up
```

## Screenshots

From the repo root (API + `npm run dev` in `web`):

```bash
SCREENSHOT_BASE_URL=http://localhost:3000 npm run screenshots
```
