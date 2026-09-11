package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mide7/go-financial-ledger-api/internal/services"
)

type IHandler interface {
	// Health
	HealthCheck(w http.ResponseWriter, r *http.Request)
	// Accounts
	CreateAccount(w http.ResponseWriter, r *http.Request)
	GetAccountDetails(w http.ResponseWriter, r *http.Request)
	ListAccounts(w http.ResponseWriter, r *http.Request)
	GetAccountBalance(w http.ResponseWriter, r *http.Request)
	GetAccountStatement(w http.ResponseWriter, r *http.Request)
	GetAccountEntries(w http.ResponseWriter, r *http.Request)

	// Transactions
	CreateTransaction(w http.ResponseWriter, r *http.Request)
	GetTransactionDetails(w http.ResponseWriter, r *http.Request)
	ListTransactions(w http.ResponseWriter, r *http.Request)
	ReverseTransaction(w http.ResponseWriter, r *http.Request)
	GetTransactionEntries(w http.ResponseWriter, r *http.Request)
}

type Handler struct {
	accountService     services.IAccountService
	transactionService services.ITransactionService
	dbPool             *pgxpool.Pool
}

func NewHandler(accountService services.IAccountService, transactionService services.ITransactionService, dbPool *pgxpool.Pool) *Handler {
	return &Handler{
		accountService:     accountService,
		transactionService: transactionService,
		dbPool:             dbPool,
	}
}
