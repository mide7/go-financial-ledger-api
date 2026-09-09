package handlers

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/db"
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
}

func NewHandler(d db.Querier) *Handler {
	return &Handler{
		accountService:     services.NewAccountService(d),
		transactionService: services.NewTransactionService(d),
	}
}
