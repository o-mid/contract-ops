# Two-Minute Demo Script

1. Start the API and web app in separate terminals.
2. Open the console and show the initial event list.
3. Tab through the search field and status selector to show visible focus.
4. Search for `fireblocks`, then choose `Processed`. Explain that the API receives both filters and returns a typed event page.
5. Search for a value that has no result. Show the explicit empty state.
6. Stop the API, refresh, and show the error state with retry.
7. Open `openapi.yaml`, then `api/internal/httpapi/server.go`, and show that the handler validates filter values before querying the fixture store.
8. Run `npm run check` and `cd api && go test ./...`.

Keep the recording factual. Do not imply that the fixture service is a production system.
