// Package ready reports whether this process can serve traffic.
package ready

import (
	"context"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/o-mid/contract-ops/api/migrations"
)

// Check pings Postgres and compares the applied goose version with the
// migrations embedded in this binary. A process that is behind the schema
// should not receive traffic.
func Check(ctx context.Context, pool *pgxpool.Pool) error {
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database unreachable: %w", err)
	}

	var current int64
	if err := pool.QueryRow(ctx, `SELECT coalesce(max(version_id), 0) FROM goose_db_version`).Scan(&current); err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}

	latest, err := Latest()
	if err != nil {
		return err
	}
	if current != latest {
		return fmt.Errorf("database version %d, binary version %d", current, latest)
	}
	return nil
}

// Latest is the highest numeric prefix in the embedded migration files.
func Latest() (int64, error) {
	var latest int64
	err := fs.WalkDir(migrations.FS, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			return err
		}
		prefix, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			return nil
		}
		version, convErr := strconv.ParseInt(prefix, 10, 64)
		if convErr != nil {
			return nil
		}
		if version > latest {
			latest = version
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("read migrations: %w", err)
	}
	if latest == 0 {
		return 0, fmt.Errorf("no migrations embedded")
	}
	return latest, nil
}
