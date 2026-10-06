package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-chi/chi/v5"

	"github.com/o-mid/contract-ops/api/internal/activity"
	"github.com/o-mid/contract-ops/api/internal/connections"
	"github.com/o-mid/contract-ops/api/internal/connectors"
	"github.com/o-mid/contract-ops/api/internal/connectors/fakevendor"
	"github.com/o-mid/contract-ops/api/internal/credentials"
	"github.com/o-mid/contract-ops/api/internal/events"
	"github.com/o-mid/contract-ops/api/internal/httpapi"
	"github.com/o-mid/contract-ops/api/internal/platform/auth"
	"github.com/o-mid/contract-ops/api/internal/platform/config"
	"github.com/o-mid/contract-ops/api/internal/platform/db"
	"github.com/o-mid/contract-ops/api/internal/platform/idempotency"
	applog "github.com/o-mid/contract-ops/api/internal/platform/log"
	"github.com/o-mid/contract-ops/api/internal/platform/migrate"
	"github.com/o-mid/contract-ops/api/internal/platform/ready"
	"github.com/o-mid/contract-ops/api/internal/platform/telemetry"
	"github.com/o-mid/contract-ops/api/internal/sync"
)

// main migrates before it serves. stopListen is deferred after pool.Close,
// and defers run in reverse, so LISTEN finishes while the pool is still open.
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

	keys := auth.NewStore(pool)
	bootstrapKey, err := config.BootstrapAPIKey()
	if err != nil {
		logger.Error("invalid config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := keys.EnsureBootstrap(ctx, bootstrapKey); err != nil {
		logger.Error("bootstrap api key", slog.String("error", err.Error()))
		os.Exit(1)
	}

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

	masterKey, err := config.MasterKey()
	if err != nil {
		logger.Error("invalid config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	keyProvider, err := credentials.NewEnvKey(masterKey)
	if err != nil {
		logger.Error("invalid config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	registry := connectors.NewRegistry(
		fakevendor.New(),
		connectors.Disabled{KindName: "openai", DisplayName: "OpenAI"},
		connectors.Disabled{KindName: "anthropic", DisplayName: "Anthropic"},
	)
	connectionStore := connections.NewStore(pool)
	connectionHandler := connections.NewHandler(connections.NewService(
		connectionStore,
		credentials.NewSealer(keyProvider),
		registryVerifier{registry: registry},
	))
	syncHandler := sync.NewHandler(sync.NewStore(pool), connectionStore)

	metrics := telemetry.New()
	server := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewServer(store, httpapi.Options{
			Logger:     logger,
			CORSOrigin: cfg.CORSOrigin,
			Ready: func(ctx context.Context) error {
				return ready.Check(ctx, pool)
			},
			Metrics:      metrics.Handler(),
			Instrument:   metrics.Middleware,
			Authenticate: auth.Middleware(keys.Resolve),
			Idempotency:  idempotency.Middleware(idempotency.NewStore(pool)),
			Register: func(router chi.Router) {
				connectionHandler.Routes(router)
				syncHandler.Routes(router)
			},
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

type registryVerifier struct {
	registry *connectors.Registry
}

func (v registryVerifier) Verify(ctx context.Context, kind, secret string) error {
	connector, ok := v.registry.Get(kind)
	if !ok {
		return &connections.Failure{Status: http.StatusBadRequest, Detail: "unknown connector"}
	}
	if _, disabled := connector.(connectors.Disabled); disabled {
		return &connections.Failure{Status: http.StatusBadRequest, Detail: "connector is not enabled"}
	}
	if err := connector.Verify(ctx, connectors.Credential{Secret: secret}); err != nil {
		return &connections.Failure{Status: http.StatusUnauthorized, Code: "auth_invalid", Detail: "credential was rejected"}
	}
	return nil
}
