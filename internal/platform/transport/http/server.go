package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middlewares"
)

type HttpServer struct {
	server *http.Server
}

func NewHttpServer(port string, router http.Handler) *HttpServer {
	addr := port
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	middlewareStack := middlewares.Chain(
		middlewares.Recovery,
		middlewares.Logging,
		middlewares.JSONResponse,
	)

	return &HttpServer{

		server: &http.Server{
			Addr:         addr,
			Handler:      middlewareStack(router),
			WriteTimeout: 30 * time.Second,
			ReadTimeout:  10 * time.Second,
			IdleTimeout:  time.Minute,
		},
	}
}

func (s *HttpServer) Start() error {
	slog.Info("🚀 starting HTTP server", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *HttpServer) Stop(ctx context.Context) error {
	slog.Warn("⏳ shutting down HTTP server...")
	return s.server.Shutdown(ctx)
}
