// Package migrate applies the embedded goose migrations.
package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/o-mid/contract-ops/api/migrations"
	"github.com/pressly/goose/v3"
)

// Up applies every pending migration.
func Up(ctx context.Context, db *sql.DB) error {
	if err := prepare(db); err != nil {
		return err
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// Down rolls back the latest migration. The SQL files are written so this is safe to re-run in CI.
func Down(ctx context.Context, db *sql.DB) error {
	if err := prepare(db); err != nil {
		return err
	}
	if err := goose.DownContext(ctx, db, "."); err != nil {
		return fmt.Errorf("migrate down: %w", err)
	}
	return nil
}

func prepare(db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	return nil
}
