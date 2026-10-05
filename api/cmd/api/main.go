package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/o-mid/contract-ops/api/internal/events"
	"github.com/o-mid/contract-ops/api/internal/httpapi"
	"github.com/o-mid/contract-ops/api/internal/platform/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	store := events.NewStore(events.Fixtures())
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewServer(store).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.DemoGenerator {
		go events.RunGenerator(ctx, store, 5*time.Second)
	}

	go func() {
		log.Printf("contract-ops API listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("contract-ops API shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}
