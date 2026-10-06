package sync

import "time"

// Chunks splits a backfill into day-sized windows. The caller runs them one
// at a time per connection.
func Chunks(start, end time.Time) [][2]time.Time {
	start = start.UTC()
	end = end.UTC()
	if !start.Before(end) {
		return nil
	}
	var out [][2]time.Time
	for cursor := start; cursor.Before(end); {
		next := cursor.Add(24 * time.Hour)
		if next.After(end) {
			next = end
		}
		out = append(out, [2]time.Time{cursor, next})
		cursor = next
	}
	return out
}

func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	wait := time.Second << min(attempt, 8)
	if wait > 5*time.Minute {
		wait = 5 * time.Minute
	}
	return wait
}
