# Two-minute demo

Use this when walking through the console. Connections, sealed credentials, and the sync worker are in the API. This script stays on the screen a reviewer can see.

1. Start the stack with `make up`, then `npm run dev`.
2. Open `http://localhost:5173` and show the initial event list.
3. Leave the page open. A new event appears about every five seconds until eight are stored. The "Last updated" line changes without a refresh.
4. Tab through the search field and the status select. Focus is visible.
5. Search for `fireblocks`, then choose `Processed`. The SSE URL carries both filters. Each push is one event page.
6. Search for a value that matches nothing. The empty state is its own message, not a blank table.
7. Stop the API. The stream error offers a retry.
8. Open `openapi.yaml`, then `api/internal/httpapi/server.go`, and show the stream handler checking `status` before it reads the store.
9. If the conversation moves past the console, open `docs/architecture.md` at the sync section: one running job per connection, and a drifted page that does not move the cursor.
10. Run `npm run check` and `cd api && go test ./...`.

The generator stops at eight fixtures. It is not a vendor sync. Screenshots are in [`screenshots/`](./screenshots/).
