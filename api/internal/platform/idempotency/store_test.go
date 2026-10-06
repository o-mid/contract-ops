package idempotency

import (
	"bytes"
	"context"
	"database/sql"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/o-mid/contract-ops/api/internal/platform/auth"
	"github.com/o-mid/contract-ops/api/internal/platform/db"
	"github.com/o-mid/contract-ops/api/internal/platform/migrate"
)

func TestStoreReplaysAcrossCalls(t *testing.T) {
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool := openPool(t, base)
	if err := auth.NewStore(pool).EnsureBootstrap(ctx, "co_local_dev_key_not_for_production"); err != nil {
		t.Fatal(err)
	}

	store := NewStore(pool)
	hash := []byte("hash-1")
	if _, claimed, err := store.Claim(ctx, auth.LocalWorkspaceID, "req_1", hash); err != nil || !claimed {
		t.Fatalf("claim = %v %v", claimed, err)
	}
	if err := store.Complete(ctx, auth.LocalWorkspaceID, "req_1", 201, []byte(`{"ok":true}`), "application/json"); err != nil {
		t.Fatal(err)
	}

	record, claimed, err := store.Claim(ctx, auth.LocalWorkspaceID, "req_1", hash)
	if err != nil || claimed {
		t.Fatalf("second claim = %v %v", claimed, err)
	}
	if record.Status != 201 || !bytes.Equal(record.Body, []byte(`{"ok":true}`)) {
		t.Fatalf("record = %+v", record)
	}
}

func openPool(t *testing.T, base string) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	admin, err := sql.Open("pgx", base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.Background(), `DROP DATABASE IF EXISTS contract_ops_idem_test WITH (FORCE)`)
		_ = admin.Close()
	})
	if err := admin.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	_, _ = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS contract_ops_idem_test WITH (FORCE)`)
	if _, err := admin.ExecContext(ctx, `CREATE DATABASE contract_ops_idem_test`); err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/contract_ops_idem_test"

	sqlDB, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := migrate.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}

	pool, err := db.Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
