package http

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/handlers"
)

func NewRouter(h handlers.Handler, healthCheckHandler handlers.HealthCheckHandler) http.Handler {
	router := http.NewServeMux()

	healthHandler := RegisterHealthRoutes(healthCheckHandler)
	router.Handle("/health/", http.StripPrefix("/health", healthHandler))

	v1Handler := RegisterV1Routes(h)
	router.Handle("/api/v1/", http.StripPrefix("/api/v1", v1Handler))

	return router
}

func RegisterHealthRoutes(h handlers.HealthCheckHandler) http.Handler {
	router := http.NewServeMux()

	// Health
	router.HandleFunc("GET /", h.HealthCheck)

	return router
}

func RegisterV1Routes(h handlers.Handler) http.Handler {
	router := http.NewServeMux()

	// Accounts
	router.HandleFunc("POST /accounts", h.CreateAccount)
	router.HandleFunc("GET /accounts/{id}", h.GetAccountDetails)
	router.HandleFunc("GET /accounts", h.ListAccounts)
	router.HandleFunc("GET /accounts/{id}/balance", h.GetAccountBalance)
	router.HandleFunc("GET /accounts/{id}/statement", h.GetAccountStatement)
	router.HandleFunc("GET /accounts/{id}/entries", h.GetAccountEntries)
	router.HandleFunc("POST /accounts/{id}/snapshot", h.CreateAccountSnapshot)
	router.HandleFunc("POST /accounts/{id}/reconcile", h.ReconcileAccount)

	// Transactions
	router.HandleFunc("POST /transactions", h.CreateTransaction)
	router.HandleFunc("GET /transactions/{id}", h.GetTransactionDetails)
	router.HandleFunc("GET /transactions", h.ListTransactions)
	router.HandleFunc("POST /transactions/{id}/reverse", h.ReverseTransaction)

	return router
}
