# Two-Minute Demo Script

Use this when walking through the repository in an interview or portfolio review.

1. Start the API and web app in separate terminals.
2. Open the console and show the initial event list.
3. Leave the page open and watch a new event appear about every five seconds until eight events are shown. Point out the Last updated timestamp changing without a manual refresh.
4. Tab through the search field and status selector to show visible focus.
5. Search for `fireblocks`, then choose `Processed`. Explain that the SSE URL carries both filters and the API returns a typed event page on each push.
6. Search for a value that has no result. Show the explicit empty state.
7. Stop the API and show the stream error state with retry.
8. Open `openapi.yaml`, then `api/internal/httpapi/server.go`, and show the stream handler validating filters before reading the store.
9. Run `npm run check` and `cd api && go test ./...`.

Keep the recording factual. Do not imply that the fixture generator is a production event pipeline.

Reference screenshots live in [`screenshots/`](./screenshots/).
