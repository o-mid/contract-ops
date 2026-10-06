package scaffold

import (
	"context"
	"testing"

	"github.com/o-mid/contract-ops/api/internal/connectors"
)

func TestScaffoldFetchesAnEmptyPage(t *testing.T) {
	connector := New()
	if connector.Kind() != "scaffold" {
		t.Fatalf("kind = %s", connector.Kind())
	}
	page, err := connector.Fetch(context.Background(), connectors.Credential{}, connectors.Window{}, connectors.Cursor{})
	if err != nil {
		t.Fatal(err)
	}
	if !page.Done || len(page.Records) != 0 {
		t.Fatalf("page = %+v", page)
	}
}
