// Package handler implements the HTTP handlers for the ledger API.
package handler

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/httputil"
)

// CreateAccount opens an account from the JSON body.
func (h *handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAccountDTO

	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		h.fail(w, r, err)
		return
	}

	account, err := h.accountService.CreateAccount(r.Context(), req.ToDomainParams())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSONMsg(w, http.StatusCreated, "account created", account)
}

// GetAccountDetails returns the account named in the path.
func (h *handler) GetAccountDetails(w http.ResponseWriter, r *http.Request) {
	var req dto.GetAccountDetailsDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath); err != nil {
		h.fail(w, r, err)
		return
	}

	account, err := h.accountService.GetAccountDetails(r.Context(), req.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSON(w, http.StatusOK, account)
}

// ListAccounts returns a page of accounts filtered by the query string.
func (h *handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	var req dto.ListAccountsDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourceQuery); err != nil {
		h.fail(w, r, err)
		return
	}

	accounts, err := h.accountService.ListAccounts(r.Context(), req.ToDomainParams())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSON(w, http.StatusOK, accounts)
}

// GetAccountBalance returns the balance of an account.
func (h *handler) GetAccountBalance(w http.ResponseWriter, r *http.Request) {
	var req dto.GetAccountBalanceDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath); err != nil {
		h.fail(w, r, err)
		return
	}

	balance, err := h.accountService.GetAccountBalance(r.Context(), req.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSON(w, http.StatusOK, balance)
}

// GetAccountStatementSummary returns an account's statement summary for the
// start_date and end_date query parameters.
func (h *handler) GetAccountStatementSummary(w http.ResponseWriter, r *http.Request) {
	var req dto.GetAccountStatementSummaryDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath, httputil.SourceQuery); err != nil {
		h.fail(w, r, err)
		return
	}

	statement, err := h.accountService.GetAccountStatementSummary(r.Context(), req.ToDomainParams())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSON(w, http.StatusOK, statement)
}

// GetAccountStatementEntries returns a cursor page of an account's statement
// entries for the date range in the query string.
func (h *handler) GetAccountStatementEntries(w http.ResponseWriter, r *http.Request) {
	var req dto.GetAccountStatementEntriesDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath, httputil.SourceQuery); err != nil {
		h.fail(w, r, err)
		return
	}

	entries, err := h.accountService.GetAccountStatementEntries(r.Context(), req.ToDomainParams())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSON(w, http.StatusOK, entries)
}

// ReconcileAccount compares the account's ledger with its latest snapshot.
func (h *handler) ReconcileAccount(w http.ResponseWriter, r *http.Request) {
	var req dto.ReconcileAccountDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath, httputil.SourceBody); err != nil {
		h.fail(w, r, err)
		return
	}

	result, err := h.accountService.ReconcileAccount(r.Context(), req.ID)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSONMsg(w, http.StatusOK, "account reconciled", result)
}

// CreateAccountSnapshot checkpoints the account's current balance.
func (h *handler) CreateAccountSnapshot(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAccountSnapshotDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath, httputil.SourceBody); err != nil {
		h.fail(w, r, err)
		return
	}

	snapshot, err := h.accountService.CreateAccountSnapshot(r.Context(), req.ToDomainParams())
	if err != nil {
		h.fail(w, r, err)
		return
	}

	httputil.JSONMsg(w, http.StatusCreated, "account snapshot created", snapshot)
}
