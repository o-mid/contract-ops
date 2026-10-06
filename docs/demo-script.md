# Two-minute demo

Use this when walking through the console. The API still owns credentials, sealing, and sync; this script focuses on what reviewers see in the browser.

1. Start the stack with `make up`, then `npm run dev`.
2. Open `http://localhost:3000` (landing) and `http://localhost:3000/console`. Show **Activity**: sidebar, live indicator, and the initial event table.
3. Leave the page open. A new event appears about every five seconds until eight are stored. The “Last updated” line changes without a refresh.
4. Tab through **Search** and **Status**. Focus rings stay visible on the dark theme.
5. Search for `fireblocks`, then choose **Processed**. The URL updates (`?q=…&status=processed`); the stream reconnects with those query params.
6. Search for a value that matches nothing. The empty state is explicit, not a blank table.
7. Open **Settings**, paste the local key from `.env.example` (`BOOTSTRAP_API_KEY`), save.
8. Open **Connections** (two demo vendors are already seeded after a fresh volume). Optionally create another `fakevendor`, **Verify**, then **Backfill 24h**. Mention jobs are API-only until a sync list endpoint exists.
9. Stop the API. The stream error offers **Try again**.
10. Optional depth: `openapi.yaml`, `api/internal/httpapi/server.go` (stream filters), `docs/architecture.md` (sync leases).
11. Run `npm run check` and `cd api && go test ./...`.

The generator stops at eight fixtures. It is not a vendor sync.

Screenshots: [`screenshots/`](./screenshots/) (see `npm run screenshots` in the root `package.json`).
