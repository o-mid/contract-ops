package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/o-mid/contract-ops/api/internal/connections"
	"github.com/o-mid/contract-ops/api/internal/connectors"
	"github.com/o-mid/contract-ops/api/internal/connectors/fakevendor"
	"github.com/o-mid/contract-ops/api/internal/credentials"
	"github.com/o-mid/contract-ops/api/internal/platform/config"
	"github.com/o-mid/contract-ops/api/internal/platform/db"
	applog "github.com/o-mid/contract-ops/api/internal/platform/log"
	"github.com/o-mid/contract-ops/api/internal/platform/migrate"
	"github.com/o-mid/contract-ops/api/internal/sync"
)

// main migrates, then runs one scheduler and WORKER_CONCURRENCY runners.
// The scheduler lock decides who enqueues. Runners only claim.
func main() {
	logger := applog.New(slog.LevelInfo)
	slog.SetDefault(logger)

	databaseURL, err := config.DatabaseURL()
	if err != nil {
		logger.Error("invalid config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	masterKey, err := config.MasterKey()
	if err != nil {
		logger.Error("invalid config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	keys, err := credentials.NewEnvKey(masterKey)
	if err != nil {
		logger.Error("invalid config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := applyMigrations(ctx, databaseURL); err != nil {
		logger.Error("migrate", slog.String("error", err.Error()))
		os.Exit(1)
	}
	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		logger.Error("database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	registry := connectors.NewRegistry(
		fakevendor.New(),
		connectors.Disabled{KindName: "openai", DisplayName: "OpenAI"},
		connectors.Disabled{KindName: "anthropic", DisplayName: "Anthropic"},
	)
	jobs := sync.NewStore(pool)
	runner := &sync.Runner{
		Jobs:        jobs,
		Connections: connections.NewStore(pool),
		Sealer:      credentials.NewSealer(keys),
		Registry:    registry,
		Lease:       30 * time.Second,
		Lookback:    24 * time.Hour,
	}
	scheduler := &sync.Scheduler{Store: jobs, Pool: pool, Interval: time.Minute, Lookback: 24 * time.Hour}

	concurrency := 2
	if raw := os.Getenv("WORKER_CONCURRENCY"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 32 {
			logger.Error("WORKER_CONCURRENCY must be from 1 to 32")
			os.Exit(1)
		}
		concurrency = parsed
	}

	errCh := make(chan error, concurrency+1)
	go func() { errCh <- scheduler.Run(ctx) }()
	for range concurrency {
		go func() { errCh <- runner.Run(ctx) }()
	}

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("worker", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}
	logger.Info("shutting down")
}

func applyMigrations(ctx context.Context, databaseURL string) error {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return err
	}
	return migrate.Up(pingCtx, sqlDB)
}
