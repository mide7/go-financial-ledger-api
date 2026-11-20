package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/httputil"
)

// Handler serves the versioned API routes.
type Handler interface {
	// Accounts
	CreateAccount(w http.ResponseWriter, r *http.Request)
	GetAccountDetails(w http.ResponseWriter, r *http.Request)
	ListAccounts(w http.ResponseWriter, r *http.Request)
	GetAccountBalance(w http.ResponseWriter, r *http.Request)
	GetAccountStatementSummary(w http.ResponseWriter, r *http.Request)
	GetAccountStatementEntries(w http.ResponseWriter, r *http.Request)
	ReconcileAccount(w http.ResponseWriter, r *http.Request)
	CreateAccountSnapshot(w http.ResponseWriter, r *http.Request)

	// Transactions
	CreateTransaction(w http.ResponseWriter, r *http.Request)
	GetTransactionDetails(w http.ResponseWriter, r *http.Request)
	ListTransactions(w http.ResponseWriter, r *http.Request)
	ReverseTransaction(w http.ResponseWriter, r *http.Request)
}

// HealthCheckHandler serves the /health route.
type HealthCheckHandler interface {
	// Health
	HealthCheck(w http.ResponseWriter, r *http.Request)
}

type handler struct {
	accountService     account.Service
	transactionService transaction.Service
	log                *slog.Logger
}

// NewHandler returns a Handler that delegates to the given services and logs
// unexpected (500) errors to log.
func NewHandler(accountService account.Service, transactionService transaction.Service, log *slog.Logger) Handler {
	return &handler{
		accountService:     accountService,
		transactionService: transactionService,
		log:                log,
	}
}

// fail writes err to the client and logs it when it was unexpected (500). The
// client only sees a generic message then, so the log is the one place the
// cause is recorded. Known errors, including 501 for unimplemented endpoints,
// are not logged here; the request log already records their status.
func (h *handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	if status := httputil.Error(w, err); status == http.StatusInternalServerError {
		h.log.ErrorContext(r.Context(), "request failed",
			"err", err,
			"status", status,
			"method", r.Method,
			"path", r.URL.Path,
		)
	}
}

// Pinger reports whether a backing service is reachable. *pgxpool.Pool satisfies it.
type Pinger interface {
	Ping(ctx context.Context) error
}

type healthCheckHandler struct {
	db Pinger
}

// NewHealthCheckHandler returns a HealthCheckHandler that pings db.
func NewHealthCheckHandler(db Pinger) HealthCheckHandler {
	return &healthCheckHandler{
		db: db,
	}
}
