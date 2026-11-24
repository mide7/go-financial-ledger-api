// Package http wires the HTTP router, middleware and server.
package http

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handler"
)

// NewRouter mounts the health routes at /health and the API at /api/v1.
func NewRouter(h handler.Handler, healthCheckHandler handler.HealthCheckHandler) http.Handler {
	router := http.NewServeMux()

	healthHandler := RegisterHealthRoutes(healthCheckHandler)
	router.Handle("/health/", http.StripPrefix("/health", healthHandler))

	v1Handler := RegisterV1Routes(h)
	router.Handle("/api/v1/", http.StripPrefix("/api/v1", v1Handler))

	return router
}

// RegisterHealthRoutes returns a mux serving the health check.
func RegisterHealthRoutes(h handler.HealthCheckHandler) http.Handler {
	router := http.NewServeMux()

	router.HandleFunc("GET /", h.HealthCheck)

	return router
}

// RegisterV1Routes returns a mux serving the v1 account and transaction routes.
func RegisterV1Routes(h handler.Handler) http.Handler {
	router := http.NewServeMux()

	router.HandleFunc("POST /accounts", h.CreateAccount)
	router.HandleFunc("GET /accounts", h.ListAccounts)
	router.HandleFunc("GET /accounts/{id}/balance", h.GetAccountBalance)
	router.HandleFunc("POST /accounts/{id}/reconcile", h.ReconcileAccount)
	router.HandleFunc("GET /accounts/{id}/statement/summary", h.GetAccountStatementSummary)
	router.HandleFunc("GET /accounts/{id}/statement/entries", h.GetAccountStatementEntries)
	router.HandleFunc("POST /accounts/{id}/snapshot", h.CreateAccountSnapshot)
	router.HandleFunc("GET /accounts/{id}", h.GetAccountDetails)

	router.HandleFunc("POST /transactions", h.CreateTransaction)
	router.HandleFunc("GET /transactions", h.ListTransactions)
	router.HandleFunc("POST /transactions/{id}/reverse", h.ReverseTransaction)
	router.HandleFunc("GET /transactions/{id}", h.GetTransactionDetails)

	return router
}
