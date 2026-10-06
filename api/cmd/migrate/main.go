package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/o-mid/contract-ops/api/internal/platform/config"
	applog "github.com/o-mid/contract-ops/api/internal/platform/log"
	"github.com/o-mid/contract-ops/api/internal/platform/migrate"
)

func main() {
	logger := applog.New(slog.LevelInfo)

	databaseURL, err := config.DatabaseURL()
	if err != nil {
		logger.Error("invalid config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		logger.Error("open database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("close database", slog.String("error", err.Error()))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		logger.Error("ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}

	switch direction {
	case "up":
		err = migrate.Up(ctx, db)
	case "down":
		err = migrate.Down(ctx, db)
	default:
		logger.Error("usage: migrate [up|down]")
		os.Exit(2)
	}
	if err != nil {
		logger.Error("migrate", slog.String("direction", direction), slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("migrations applied", slog.String("direction", direction))
}
