package events

import (
	"context"
	"testing"
	"time"
)

func TestRunGeneratorAppendsUntilMaxEvents(t *testing.T) {
	store := NewStore(Fixtures())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		RunGenerator(ctx, store, 5*time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("generator did not finish")
	}

	if store.Len() != MaxEvents {
		t.Fatalf("expected %d events, got %d", MaxEvents, store.Len())
	}

	page := store.List("", "")
	ids := map[string]bool{}
	for _, event := range page.Events {
		ids[event.ID] = true
	}
	for _, pending := range PendingFixtures() {
		if !ids[pending.ID] {
			t.Fatalf("missing pending fixture %q", pending.ID)
		}
	}
}

func TestRunGeneratorStopsOnCancel(t *testing.T) {
	store := NewStore(Fixtures())
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		RunGenerator(ctx, store, time.Hour)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("generator did not stop after cancel")
	}

	if store.Len() != len(Fixtures()) {
		t.Fatalf("expected no appends before first tick, got %d", store.Len())
	}
}
