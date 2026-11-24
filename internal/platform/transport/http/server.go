package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middleware"
)

// Server is the API's HTTP server with its middleware stack applied.
type Server struct {
	server *http.Server
	log    *slog.Logger
}

// NewServer returns a server listening on port (a leading colon is optional).
// Request logs, recovered panics and net/http's own errors all go to log.
func NewServer(port string, router http.Handler, log *slog.Logger) *Server {
	addr := port
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	middlewareStack := middleware.Chain(
		middleware.Recovery(log),
		middleware.Logging(log),
		middleware.JSONResponse,
	)

	return &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      middlewareStack(router),
			WriteTimeout: 30 * time.Second,
			ReadTimeout:  10 * time.Second,
			IdleTimeout:  time.Minute,
			ErrorLog:     slog.NewLogLogger(log.Handler(), slog.LevelError),
		},
		log: log,
	}
}

// Start listens and serves until the server is stopped. It returns http.ErrServerClosed after Stop.
func (s *Server) Start() error {
	s.log.Info("🚀 starting HTTP server", "addr", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the server, waiting for active requests until ctx is done.
func (s *Server) Stop(ctx context.Context) error {
	s.log.InfoContext(ctx, "⏳ shutting down HTTP server")
	return s.server.Shutdown(ctx)
}
