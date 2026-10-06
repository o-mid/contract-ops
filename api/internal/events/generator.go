package events

import (
	"context"
	"time"
)

// RunGenerator appends pending fixtures on interval until MaxEvents or ctx cancel.
func RunGenerator(ctx context.Context, store Feed, interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	pending := PendingFixtures()
	index := 0

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			count, err := store.Len(ctx)
			if err != nil || count >= MaxEvents || index >= len(pending) {
				return
			}

			event := pending[index]
			event.OccurredAt = now.UTC()
			inserted, err := store.Append(ctx, event)
			if err != nil {
				return
			}
			index++
			// A duplicate id is already stored. Keep walking the fixture list.
			if !inserted && count >= MaxEvents {
				return
			}
		}
	}
}
