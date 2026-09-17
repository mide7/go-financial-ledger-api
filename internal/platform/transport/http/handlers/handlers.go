package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
)

type Handler interface {
	// Accounts
	CreateAccount(w http.ResponseWriter, r *http.Request)
	GetAccountDetails(w http.ResponseWriter, r *http.Request)
	ListAccounts(w http.ResponseWriter, r *http.Request)
	GetAccountBalance(w http.ResponseWriter, r *http.Request)
	GetAccountStatement(w http.ResponseWriter, r *http.Request)
	GetAccountEntries(w http.ResponseWriter, r *http.Request)
	ReconcileAccount(w http.ResponseWriter, r *http.Request)
	CreateAccountSnapshot(w http.ResponseWriter, r *http.Request)

	// Transactions
	CreateTransaction(w http.ResponseWriter, r *http.Request)
	GetTransactionDetails(w http.ResponseWriter, r *http.Request)
	ListTransactions(w http.ResponseWriter, r *http.Request)
	ReverseTransaction(w http.ResponseWriter, r *http.Request)
}

type HealthCheckHandler interface {
	// Health
	HealthCheck(w http.ResponseWriter, r *http.Request)
}

type handler struct {
	accountService     account.Service
	transactionService transaction.Service
}

func NewHandler(accountService account.Service, transactionService transaction.Service) Handler {
	return &handler{
		accountService:     accountService,
		transactionService: transactionService,
	}
}

type healthCheckHandler struct {
	dbPool *pgxpool.Pool
}

func NewHealthCheckHandler(dbPool *pgxpool.Pool) HealthCheckHandler {
	return &healthCheckHandler{
		dbPool: dbPool,
	}
}
