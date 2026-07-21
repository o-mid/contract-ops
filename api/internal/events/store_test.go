package events

import (
	"testing"
	"time"
)

func TestStoreListFiltersByStatusAndQuery(t *testing.T) {
	store := NewStore(Fixtures())

	page := store.List("fireblocks", string(StatusProcessed))

	if page.Total != 1 {
		t.Fatalf("expected one event, got %d", page.Total)
	}

	if page.Events[0].ID != "evt_01HV1B01" {
		t.Fatalf("expected fireblocks event, got %q", page.Events[0].ID)
	}
}

func TestStoreListOrdersEventsMostRecentFirst(t *testing.T) {
	store := NewStore(Fixtures())

	page := store.List("", "")

	if page.Events[0].ID != "evt_01HV1B00" {
		t.Fatalf("expected newest event first, got %q", page.Events[0].ID)
	}
}

func TestStoreAppendCapsAtMaxEvents(t *testing.T) {
	store := NewStore(Fixtures())

	for _, pending := range PendingFixtures() {
		pending.OccurredAt = time.Now().UTC()
		if !store.Append(pending) {
			t.Fatalf("expected append to succeed for %q", pending.ID)
		}
	}

	if store.Len() != MaxEvents {
		t.Fatalf("expected %d events, got %d", MaxEvents, store.Len())
	}

	extra := Event{
		ID:            "evt_overflow",
		Source:        "test",
		Type:          "should.not.append",
		Status:        StatusPending,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "crl_overflow",
	}
	if store.Append(extra) {
		t.Fatal("expected append beyond MaxEvents to fail")
	}
	if store.Len() != MaxEvents {
		t.Fatalf("expected length to remain %d, got %d", MaxEvents, store.Len())
	}
}

func TestStoreAppendNotifiesSubscribers(t *testing.T) {
	store := NewStore(Fixtures())
	changes, cancel := store.Subscribe()
	defer cancel()

	pending := PendingFixtures()[0]
	pending.OccurredAt = time.Now().UTC()
	if !store.Append(pending) {
		t.Fatal("expected append to succeed")
	}

	select {
	case <-changes:
	case <-time.After(time.Second):
		t.Fatal("expected subscriber notification")
	}
}

func TestStoreAppendKeepsNewestFirst(t *testing.T) {
	store := NewStore(Fixtures())
	pending := PendingFixtures()[0]
	pending.OccurredAt = time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)

	if !store.Append(pending) {
		t.Fatal("expected append to succeed")
	}

	page := store.List("", "")
	if page.Events[0].ID != pending.ID {
		t.Fatalf("expected appended event first, got %q", page.Events[0].ID)
	}
}
