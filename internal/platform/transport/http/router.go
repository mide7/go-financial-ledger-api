// Package http wires the HTTP router, middleware and server.
package http

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handler"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middleware"
)

// NewRouter mounts the health routes at /health and the API at /api/v1.
// idempotent wraps the routes that move money (see RegisterV1Routes); nil
// leaves them unwrapped.
func NewRouter(
	h handler.Handler,
	healthCheckHandler handler.HealthCheckHandler,
	idempotent middleware.Middleware,
) http.Handler {
	router := http.NewServeMux()

	healthHandler := RegisterHealthRoutes(healthCheckHandler)
	router.Handle("/health/", http.StripPrefix("/health", healthHandler))

	v1Handler := RegisterV1Routes(h, idempotent)
	router.Handle("/api/v1/", http.StripPrefix("/api/v1", v1Handler))

	return router
}

// RegisterHealthRoutes returns a mux serving the health check.
func RegisterHealthRoutes(h handler.HealthCheckHandler) http.Handler {
	router := http.NewServeMux()

	router.HandleFunc("GET /", h.HealthCheck)

	return router
}

// RegisterV1Routes returns a mux serving the v1 routes; docs/api.md lists them.
// The POST routes that move money are wrapped in idempotent, normally
// middleware.Idempotency, so they honour the Idempotency-Key header; nil leaves
// them unwrapped.
func RegisterV1Routes(h handler.Handler, idempotent middleware.Middleware) http.Handler {
	router := http.NewServeMux()

	if idempotent == nil {
		idempotent = func(next http.Handler) http.Handler { return next }
	}

	moneyRoute := func(pattern string, fn http.HandlerFunc) {
		router.Handle(pattern, idempotent(fn))
	}

	router.HandleFunc("POST /accounts", h.CreateAccount)
	router.HandleFunc("GET /accounts", h.ListAccounts)
	router.HandleFunc("GET /accounts/{id}/balance", h.GetAccountBalance)
	router.HandleFunc("POST /accounts/{id}/reconcile", h.ReconcileAccount)
	router.HandleFunc("GET /accounts/{id}/statement/summary", h.GetAccountStatementSummary)
	router.HandleFunc("GET /accounts/{id}/statement/entries", h.GetAccountStatementEntries)
	router.HandleFunc("POST /accounts/{id}/snapshot", h.CreateAccountSnapshot)
	router.HandleFunc("GET /accounts/{id}/snapshot", h.GetLatestAccountSnapshot)
	router.HandleFunc("PATCH /accounts/{id}/status", h.ChangeAccountStatus)
	router.HandleFunc("GET /accounts/{id}/status/history", h.ListAccountStatusHistory)
	router.HandleFunc("GET /accounts/{id}", h.GetAccountDetails)

	moneyRoute("POST /transactions", h.CreateTransaction)
	router.HandleFunc("GET /transactions", h.ListTransactions)
	moneyRoute("POST /transactions/{id}/reverse", h.ReverseTransaction)
	router.HandleFunc("GET /transactions/{id}", h.GetTransactionDetails)

	moneyRoute("POST /conversions", h.CreateConversion)

	moneyRoute("POST /withdrawals", h.CreateWithdrawal)
	router.HandleFunc("GET /withdrawals", h.ListWithdrawals)
	router.HandleFunc("GET /withdrawals/{id}", h.GetWithdrawal)
	router.HandleFunc("POST /withdrawals/{id}/settle", h.SettleWithdrawal)
	router.HandleFunc("POST /withdrawals/{id}/fail", h.FailWithdrawal)

	router.HandleFunc("POST /reconciliations", h.ReconcileAllAccounts)

	router.HandleFunc("GET /discrepancies", h.ListUnresolvedDiscrepancies)
	router.HandleFunc("POST /discrepancies/{id}/resolve", h.ResolveDiscrepancy)

	router.HandleFunc("GET /currencies", h.ListCurrencies)

	return router
}
