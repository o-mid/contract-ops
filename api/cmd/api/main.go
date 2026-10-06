package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/o-mid/contract-ops/api/internal/activity"
	"github.com/o-mid/contract-ops/api/internal/events"
	"github.com/o-mid/contract-ops/api/internal/httpapi"
	"github.com/o-mid/contract-ops/api/internal/platform/config"
	"github.com/o-mid/contract-ops/api/internal/platform/db"
	applog "github.com/o-mid/contract-ops/api/internal/platform/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).Error("invalid config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger := applog.New(cfg.LogLevel)
	slog.SetDefault(logger)

	databaseURL, err := config.DatabaseURL()
	if err != nil {
		logger.Error("invalid config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, databaseURL)
	if err != nil {
		logger.Error("database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	store := activity.New(pool)
	stopListen, err := store.Listen(ctx)
	if err != nil {
		logger.Error("listen", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer stopListen()

	for _, fixture := range events.Fixtures() {
		if _, err := store.Append(ctx, fixture); err != nil {
			logger.Error("seed events", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	server := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewServer(store, httpapi.Options{
			Logger:     logger,
			CORSOrigin: cfg.CORSOrigin,
		}).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	if cfg.DemoGenerator {
		go events.RunGenerator(ctx, store, 5*time.Second)
	}

	go func() {
		logger.Info("listening", slog.String("addr", ":"+cfg.Port))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
