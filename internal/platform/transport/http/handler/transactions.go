package handler

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/httputil"
)

// CreateTransaction posts a balanced transaction from the JSON body.
func (h *handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTransactionDTO

	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		h.fail(w, r, err)
		return
	}

	params, err := req.ToDomainParams()
	if err != nil {
		h.fail(w, r, err)
		return
	}

	created, err := h.transactionService.CreateTransaction(r.Context(), params)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSONMsg(w, http.StatusCreated, "transaction created", created)
}

// GetTransactionDetails returns the transaction named in the path.
func (h *handler) GetTransactionDetails(w http.ResponseWriter, r *http.Request) {
	var req dto.GetTransactionDetailsDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath); err != nil {
		h.fail(w, r, err)
		return
	}

	txn, err := h.transactionService.GetTransactionDetails(r.Context(), req.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSON(w, http.StatusOK, txn)
}

// ListTransactions returns a page of transactions filtered by the query string.
func (h *handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	var req dto.ListTransactionsDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourceQuery); err != nil {
		h.fail(w, r, err)
		return
	}

	txns, err := h.transactionService.ListTransactions(r.Context(), req.ToDomainParams())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSON(w, http.StatusOK, txns)
}

// ReverseTransaction reverses a transaction.
func (h *handler) ReverseTransaction(w http.ResponseWriter, r *http.Request) {
	var req dto.ReverseTransactionDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath); err != nil {
		h.fail(w, r, err)
		return
	}

	if err := h.transactionService.ReverseTransaction(r.Context(), req.ID); err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSONMsg(w, http.StatusOK, "transaction reversed", nil)
}
