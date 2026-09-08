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

	"github.com/mide7/go-financial-ledger-api/internal/config"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres"
	transportHttp "github.com/mide7/go-financial-ledger-api/internal/platform/transport/http"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPostgresStorage(ctx)

	if err != nil {
		os.Exit(1)
	}
	defer pool.Close()

	err = postgres.InitStorage(ctx, pool)
	if err != nil {
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	port := config.ENVS.PORT
	slog.Info("server starting on", "port", port)
	httpServer := transportHttp.NewHttpServer(port, pool)

	go func() {
		if err = httpServer.Start(httpServer.Load()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start HTTP server", "err", err)
			os.Exit(1)
		}
	}()

	sig := <-sigChan
	slog.Warn("🚨 received shutdown", "signal", sig.String())

	httpServer.Stop()
}
