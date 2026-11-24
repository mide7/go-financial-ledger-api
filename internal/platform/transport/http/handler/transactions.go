package handler

import (
	"context"
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/pagination"
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
		func(
			ctx context.Context,
			req dto.ListTransactionsDTO,
		) (pagination.CursorPaginatedResult[transaction.Transaction], error) {
			return h.transactionService.ListTransactions(ctx, req.ToDomainParams())
		})
}

// ReverseTransaction posts a reversal of the transaction in the path and
// returns the reversal.
func (h *handler) ReverseTransaction(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{sources: fromPathBody, status: http.StatusCreated, message: "transaction reversed"},
		func(ctx context.Context, req dto.ReverseTransactionDTO) (*transaction.Transaction, error) {
			return h.transactionService.ReverseTransaction(ctx, req.ToDomainParams())
		})
}

// CreateConversion converts funds between two accounts in different currencies.
func (h *handler) CreateConversion(w http.ResponseWriter, r *http.Request) {
	h.serve(w, r, endpoint{status: http.StatusCreated, message: "conversion created"},
		func(ctx context.Context, req dto.CreateConversionDTO) (*transaction.Conversion, error) {
			return h.transactionService.CreateConversion(ctx, req.ToDomainParams())
		})
}
