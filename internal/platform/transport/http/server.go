package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres"
	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/db"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handlers"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middlewares"
)

type HttpServer struct {
	port    string
	handler *handlers.Handler
	server  *http.Server
	dbPool  *pgxpool.Pool
}

func NewHttpServer(port string, pool *pgxpool.Pool) *HttpServer {
	d := db.New(pool)
	return &HttpServer{
		port:    port,
		handler: handlers.NewHandler(d),
		dbPool:  pool,
	}
}

func (s *HttpServer) Load() http.Handler {
	router := http.NewServeMux()

	v1Handler := RegisterV1Routes(s.handler)

	router.Handle("/api/v1/", http.StripPrefix("/api/v1", v1Handler))

	return router
}

func (s *HttpServer) Start(router http.Handler) error {
	addr := s.port
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	middlewareStack := middlewares.Chain(
		middlewares.Recovery,
		middlewares.Logging,
		middlewares.JSONResponse,
	)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      middlewareStack(router),
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  10 * time.Second,
		IdleTimeout:  time.Minute,
	}

	return s.server.ListenAndServe()
}

func (s *HttpServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	slog.Warn("🚨 initiating graceful shutdown sequence...")
	if s.server != nil {
		slog.Warn("⏳ shutting down HTTP server...")
		if err := s.server.Shutdown(ctx); err != nil {
			slog.Error("failed to shutdown HTTP server", "err", err)
		}
	}

	// TODO: Add for redis, scheduler, etc.
	// if s.redisClient != nil {
	// 	slog.Info("Closing Redis connection...")
	// 	if err := s.redisClient.Close(); err != nil {
	// 		slog.Info("Redis close error", "err", err)
	// 	}
	// }

	postgres.CloseStorage(s.dbPool)

	slog.Warn("✅ graceful shutdown complete")

}
