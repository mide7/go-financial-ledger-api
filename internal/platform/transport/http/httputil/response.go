package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	goValidator "github.com/go-playground/validator/v10"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middleware"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/validation"
)

type errorMapping struct {
	err     error
	status  int
	message string
}

// errorMappings lists known errors in match order; anything else becomes a 500.
var errorMappings = []errorMapping{
	// Request decoding errors
	{ErrInvalidJSON, http.StatusUnprocessableEntity, "invalid JSON body"},
	{ErrInvalidQuery, http.StatusUnprocessableEntity, "invalid query parameters"},
	{ErrInvalidPath, http.StatusUnprocessableEntity, "invalid path parameters"},

	// Account errors
	{account.ErrAccountNotFound, http.StatusNotFound, "account not found"},
	{account.ErrInvalidCurrency, http.StatusUnprocessableEntity, "invalid currency code"},
	{account.ErrInvalidType, http.StatusUnprocessableEntity, "invalid account type"},
	{account.ErrAccountExists, http.StatusConflict, "account already exists"},
	{account.ErrDiscrepancyFound, http.StatusConflict, "account balance discrepancy detected"},
	{account.ErrInvalidDateRange, http.StatusUnprocessableEntity, "start_date must not be after end_date"},
	{account.ErrSnapshotExists, http.StatusConflict, "no new entries since the last balance snapshot"},
	{account.ErrSnapshotNotFound, http.StatusNotFound, "no balance snapshot exists for this account"},

	// Transaction errors
	{transaction.ErrDuplicateReference, http.StatusUnprocessableEntity, "duplicate reference"},
	{transaction.ErrTransactionNotFound, http.StatusNotFound, "transaction not found"},
	{transaction.ErrInvalidCurrency, http.StatusUnprocessableEntity, "invalid currency code"},
	{transaction.ErrAtLeastTwoEntries, http.StatusUnprocessableEntity, "transaction must contain at least two entries"},
	{transaction.ErrUnbalancedTransaction, http.StatusUnprocessableEntity, "debits and credits must balance to zero"},
	{transaction.ErrInvalidEntryType, http.StatusUnprocessableEntity, "invalid entry type"},
	{transaction.ErrInvalidEntryAmount, http.StatusUnprocessableEntity, "entry amount must be greater than zero"},
	{transaction.ErrInvalidAccountID, http.StatusUnprocessableEntity, "entry account ID is required"},
	{
		transaction.ErrAmountOverflow,
		http.StatusUnprocessableEntity,
		"total debits or credits exceed the maximum amount",
	},

	// Endpoints without an implementation yet
	{domain.ErrNotImplemented, http.StatusNotImplemented, "not implemented yet"},
}

// SetMessage sets a custom response message for the JSONResponse middleware.
func SetMessage(w http.ResponseWriter, msg string) {
	w.Header().Set(middleware.HeaderResponseMessage, msg)
}

// JSON writes status code and encodes data as JSON for the middleware to intercept.
func JSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)

	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// JSONMsg writes status code, sets a custom response message, and encodes data.
func JSONMsg(w http.ResponseWriter, status int, msg string, v any) {
	SetMessage(w, msg)
	JSON(w, status, v)
}

// Error writes err as a client-safe response and returns the status it used.
// Known errors map to fixed statuses and messages; anything else becomes a
// generic 500 whose details the caller should log. A nil err writes nothing
// and returns 0.
func Error(w http.ResponseWriter, err error) int {
	if err == nil {
		return 0
	}

	if ve, ok := errors.AsType[goValidator.ValidationErrors](err); ok {
		JSON(w, http.StatusUnprocessableEntity, validation.FormatValidationErrors(ve))

		return http.StatusUnprocessableEntity
	}

	for _, m := range errorMappings {
		if errors.Is(err, m.err) {
			JSON(w, m.status, m.message)
			return m.status
		}
	}

	JSON(w, http.StatusInternalServerError, "an unexpected error occurred")

	return http.StatusInternalServerError
}
