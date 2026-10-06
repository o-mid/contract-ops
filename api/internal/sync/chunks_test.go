package sync

import "testing"
import "time"

func TestChunksAreDaySized(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(50 * time.Hour)
	chunks := Chunks(start, end)
	if len(chunks) != 3 {
		t.Fatalf("chunks = %d", len(chunks))
	}
	if !chunks[0][1].Equal(start.Add(24*time.Hour)) || !chunks[2][1].Equal(end) {
		t.Fatalf("chunks = %v", chunks)
	}
}
