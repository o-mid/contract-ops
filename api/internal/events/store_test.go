package events

import (
	"context"
	"testing"
	"time"
)

func TestStoreListFiltersByStatusAndQuery(t *testing.T) {
	store := NewStore(Fixtures())

	page, err := store.List(context.Background(), Query{Text: "fireblocks", Status: string(StatusProcessed)})
	if err != nil {
		t.Fatal(err)
	}

	if page.Total != 1 {
		t.Fatalf("expected one event, got %d", page.Total)
	}

	if page.Events[0].ID != "evt_01HV1B01" {
		t.Fatalf("expected fireblocks event, got %q", page.Events[0].ID)
	}
}

func TestStoreListOrdersEventsMostRecentFirst(t *testing.T) {
	store := NewStore(Fixtures())

	page, err := store.List(context.Background(), Query{})
	if err != nil {
		t.Fatal(err)
	}

	if page.Events[0].ID != "evt_01HV1B00" {
		t.Fatalf("expected newest event first, got %q", page.Events[0].ID)
	}
}

func TestStoreAppendCapsAtMaxEvents(t *testing.T) {
	store := NewStore(Fixtures())

	for _, pending := range PendingFixtures() {
		pending.OccurredAt = time.Now().UTC()
		inserted, err := store.Append(context.Background(), pending)
		if err != nil || !inserted {
			t.Fatalf("expected append to succeed for %q", pending.ID)
		}
	}

	count, err := store.Len(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != MaxEvents {
		t.Fatalf("expected %d events, got %d", MaxEvents, count)
	}

	extra := Event{
		ID:            "evt_overflow",
		Source:        "test",
		Type:          "should.not.append",
		Status:        StatusPending,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "crl_overflow",
	}
	inserted, err := store.Append(context.Background(), extra)
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Fatal("expected append beyond MaxEvents to fail")
	}
	count, err = store.Len(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != MaxEvents {
		t.Fatalf("expected length to remain %d, got %d", MaxEvents, count)
	}
}

func TestStoreAppendNotifiesSubscribers(t *testing.T) {
	store := NewStore(Fixtures())
	changes, cancel, err := store.Subscribe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	pending := PendingFixtures()[0]
	pending.OccurredAt = time.Now().UTC()
	inserted, err := store.Append(context.Background(), pending)
	if err != nil || !inserted {
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

	inserted, err := store.Append(context.Background(), pending)
	if err != nil || !inserted {
		t.Fatal("expected append to succeed")
	}

	page, err := store.List(context.Background(), Query{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Events[0].ID != pending.ID {
		t.Fatalf("expected appended event first, got %q", page.Events[0].ID)
	}
}

func TestStoreListPaginatesWithACursor(t *testing.T) {
	store := NewStore(Fixtures())

	first, err := store.List(context.Background(), Query{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Events) != 2 || first.Total != 4 || first.NextCursor == "" {
		t.Fatalf("first page = %+v", first)
	}

	second, err := store.List(context.Background(), Query{Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Events) != 2 || second.NextCursor != "" {
		t.Fatalf("second page = %+v", second)
	}
	if second.Events[0].ID == first.Events[0].ID || second.Events[0].ID == first.Events[1].ID {
		t.Fatalf("second page repeated an id from the first: %+v", second.Events)
	}
}
