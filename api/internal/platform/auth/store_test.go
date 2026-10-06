package auth

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/o-mid/contract-ops/api/internal/platform/db"
	"github.com/o-mid/contract-ops/api/internal/platform/migrate"
)

func TestBootstrapKeyResolvesToTheLocalWorkspace(t *testing.T) {
	pool := testPool(t)
	store := NewStore(pool)
	const secret = "co_local_dev_key_not_for_production"

	ctx := context.Background()
	if err := store.EnsureBootstrap(ctx, secret); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureBootstrap(ctx, secret); err != nil {
		t.Fatal(err)
	}

	workspaceID, err := store.Resolve(ctx, secret)
	if err != nil {
		t.Fatal(err)
	}
	if workspaceID != LocalWorkspaceID {
		t.Fatalf("workspace = %s", workspaceID)
	}

	if _, err := store.Resolve(ctx, secret+"-nope"); err != ErrUnknownKey {
		t.Fatalf("unknown key err = %v", err)
	}

	var keys int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM api_keys`).Scan(&keys); err != nil {
		t.Fatal(err)
	}
	if keys != 1 {
		t.Fatalf("keys = %d", keys)
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	admin, err := sql.Open("pgx", base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.Background(), `DROP DATABASE IF EXISTS contract_ops_auth_test WITH (FORCE)`)
		_ = admin.Close()
	})
	if err := admin.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	_, _ = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS contract_ops_auth_test WITH (FORCE)`)
	if _, err := admin.ExecContext(ctx, `CREATE DATABASE contract_ops_auth_test`); err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/contract_ops_auth_test"
	testURL := parsed.String()

	sqlDB, err := sql.Open("pgx", testURL)
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

	pool, err := db.Open(ctx, testURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
