package handlers

import (
	"net/http"

	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/httputil"
)

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAccountDTO

	if err := httputil.DecodeAndValidate(r, &req); err != nil {
		httputil.Error(w, err)
		return
	}

	account, err := h.accountService.CreateAccount(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSONMsg(w, http.StatusCreated, "account created", account)
}

func (h *Handler) GetAccountDetails(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	var req dto.ListAccountsDTO

	if err := httputil.DecodeAndValidate(r, &req, httputil.SourceQuery); err != nil {
		httputil.Error(w, err)
		return
	}

	accounts, err := h.accountService.ListAccounts(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, accounts)
}

// GetAccountBalance returns the balance of an account
func (h *Handler) GetAccountBalance(w http.ResponseWriter, r *http.Request) {
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
func (h *Handler) GetAccountStatement(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) GetAccountEntries(w http.ResponseWriter, r *http.Request) {
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
