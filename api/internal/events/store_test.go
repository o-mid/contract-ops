package events

import "testing"

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
