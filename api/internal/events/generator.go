package events

import (
	"context"
	"time"
)

// RunGenerator appends pending fixtures on interval until MaxEvents or ctx cancel.
func RunGenerator(ctx context.Context, store *Store, interval time.Duration) {
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
			if store.Len() >= MaxEvents || index >= len(pending) {
				return
			}

			event := pending[index]
			event.OccurredAt = now.UTC()
			if !store.Append(event) {
				return
			}
			index++

			if store.Len() >= MaxEvents {
				return
			}
		}
	}
}
