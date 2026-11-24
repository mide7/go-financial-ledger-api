package handler

import (
	"context"
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
)

// CreateTransaction posts a balanced transaction from the JSON body.
func (h *handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{status: http.StatusCreated, message: "transaction created"},
		func(ctx context.Context, req dto.CreateTransactionDTO) (*transaction.Transaction, error) {
			params, err := req.ToDomainParams()
			if err != nil {
				return nil, err
			}

			return h.transactionService.CreateTransaction(ctx, params)
		})
}

// GetTransactionDetails returns the transaction named in the path.
func (h *handler) GetTransactionDetails(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPath, status: http.StatusOK},
		func(ctx context.Context, req dto.GetTransactionDetailsDTO) (*transaction.Transaction, error) {
			return h.transactionService.GetTransactionDetails(ctx, req.ID)
		})
}

// ListTransactions returns a page of transactions filtered by the query string.
func (h *handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromQuery, status: http.StatusOK},
		func(ctx context.Context, req dto.ListTransactionsDTO) ([]transaction.Transaction, error) {
			return h.transactionService.ListTransactions(ctx, req.ToDomainParams())
		})
}

// ReverseTransaction reverses a transaction. It answers with a message and no body.
func (h *handler) ReverseTransaction(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPath, status: http.StatusOK, message: "transaction reversed"},
		func(ctx context.Context, req dto.ReverseTransactionDTO) (any, error) {
			return nil, h.transactionService.ReverseTransaction(ctx, req.ID)
		})
}
