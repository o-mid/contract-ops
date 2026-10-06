package activity

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/o-mid/contract-ops/api/internal/events"
	"github.com/o-mid/contract-ops/api/internal/platform/db"
	"github.com/o-mid/contract-ops/api/internal/platform/migrate"
)

func TestTwoListenersSeeOneInsert(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithCancel(context.Background())
	testURL, err := isolatedDatabase(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		dropIsolatedDatabase(databaseURL)
	})

	sqlDB, err := sql.Open("pgx", testURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	pool, err := db.Open(ctx, testURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE activity_events`); err != nil {
		t.Fatal(err)
	}

	left := New(pool)
	right := New(pool)
	waitLeft, err := left.Listen(ctx)
	if err != nil {
		t.Fatal(err)
	}
	waitRight, err := right.Listen(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM activity_events WHERE id = $1`, "evt_listen_once"); err != nil {
			t.Error(err)
		}
		cancel()
		waitLeft()
		waitRight()
		pool.Close()
	})

	leftChanges, leftCancel, err := left.Subscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer leftCancel()
	rightChanges, rightCancel, err := right.Subscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer rightCancel()

	event := events.Fixtures()[0]
	event.ID = "evt_listen_once"
	inserted, err := left.Append(ctx, event)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("expected insert")
	}

	waitFor(t, leftChanges)
	waitFor(t, rightChanges)

	page, err := right.List(ctx, events.Query{Text: event.ID})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Events) != 1 || page.Events[0].ID != event.ID {
		t.Fatalf("right replica page = %+v", page)
	}
}

func isolatedDatabase(ctx context.Context, baseURL string) (string, error) {
	admin, err := sql.Open("pgx", baseURL)
	if err != nil {
		return "", err
	}
	defer func() { _ = admin.Close() }()

	if err := admin.PingContext(ctx); err != nil {
		return "", err
	}
	_, _ = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS contract_ops_test WITH (FORCE)`)
	if _, err := admin.ExecContext(ctx, `CREATE DATABASE contract_ops_test`); err != nil {
		return "", err
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	parsed.Path = "/contract_ops_test"
	return parsed.String(), nil
}

func dropIsolatedDatabase(baseURL string) {
	admin, err := sql.Open("pgx", baseURL)
	if err != nil {
		return
	}
	defer func() { _ = admin.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS contract_ops_test WITH (FORCE)`)
}

func waitFor(t *testing.T, changes <-chan struct{}) {
	t.Helper()
	select {
	case <-changes:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the other replica")
	}
}
