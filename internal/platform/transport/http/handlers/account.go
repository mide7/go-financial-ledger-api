package handlers

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/httputil"
)

func (h *handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAccountDTO

	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	account, err := h.accountService.CreateAccount(r.Context(), account.CreateAccountParams{
		OwnerID:  req.OwnerID,
		Type:     req.Type,
		Currency: req.Currency,
	})
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSONMsg(w, http.StatusCreated, "account created", account)
}

func (h *handler) GetAccountDetails(w http.ResponseWriter, r *http.Request) {
	var req dto.GetAccountDetailsDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath); err != nil {
		httputil.Error(w, err)
		return
	}

	account, err := h.accountService.GetAccountDetails(r.Context(), req.ID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, account)
}

func (h *handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	var req dto.ListAccountsDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourceQuery); err != nil {
		httputil.Error(w, err)
		return
	}

	accounts, err := h.accountService.ListAccounts(r.Context(), account.ListAccountsParams{
		Page:     req.Page,
		Limit:    req.Limit,
		Currency: req.Currency,
		Type:     req.Type,
	})
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, accounts)
}

// GetAccountBalance returns the balance of an account
func (h *handler) GetAccountBalance(w http.ResponseWriter, r *http.Request) {
	var req dto.GetAccountBalanceDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath); err != nil {
		httputil.Error(w, err)
		return
	}

	balance, err := h.accountService.GetAccountBalance(r.Context(), req.ID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, balance)
}

// GetAccountStatement returns the statement of an account
func (h *handler) GetAccountStatement(w http.ResponseWriter, r *http.Request) {
	var req dto.GetAccountStatementDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath); err != nil {
		httputil.Error(w, err)
		return
	}

	statement, err := h.accountService.GetAccountStatement(r.Context(), req.ID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, statement)
}

func (h *handler) GetAccountEntries(w http.ResponseWriter, r *http.Request) {
	var req dto.GetAccountEntriesDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath); err != nil {
		httputil.Error(w, err)
		return
	}

	entries, err := h.accountService.GetAccountEntries(r.Context(), req.ID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, entries)
}

func (h *handler) ReconcileAccount(w http.ResponseWriter, r *http.Request) {
	var req dto.ReconcileAccountDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath); err != nil {
		httputil.Error(w, err)
		return
	}

	result, err := h.accountService.ReconcileAccount(r.Context(), req.ID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSONMsg(w, http.StatusOK, "account reconciled", result)
}

func (h *handler) CreateAccountSnapshot(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAccountSnapshotDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourcePath); err != nil {
		httputil.Error(w, err)
		return
	}

	result, err := h.accountService.CreateAccountSnapshot(r.Context(), account.CreateAccountSnapshotParams{
		ID:                  req.ID,
		TargetEntrySequence: req.TargetEntrySequence,
		Reason:              req.Reason,
	})
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSONMsg(w, http.StatusOK, "account snapshot created", result)
}
