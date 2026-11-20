package httputil_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/httputil"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/middleware"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/validation"
)

func TestSetMessage(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	httputil.SetMessage(rec, "account created")

	if got := rec.Header().Get(middleware.HeaderResponseMessage); got != "account created" {
		t.Errorf("message header = %q, want %q", got, "account created")
	}
}

func TestJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    any
		wantBody string
	}{
		{name: "encodes value", value: map[string]int{"n": 1}, wantBody: "{\"n\":1}\n"},
		{name: "nil writes no body", value: nil, wantBody: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := httptest.NewRecorder()
			httputil.JSON(rec, http.StatusCreated, tt.value)

			if rec.Code != http.StatusCreated {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
			}

			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestJSONMsg(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	httputil.JSONMsg(rec, http.StatusOK, "done", true)

	if rec.Code != http.StatusOK ||
		rec.Header().Get(middleware.HeaderResponseMessage) != "done" ||
		rec.Body.String() != "true\n" {
		t.Errorf("got status %d, message %q, body %q",
			rec.Code, rec.Header().Get(middleware.HeaderResponseMessage), rec.Body,
		)
	}
}

// TestError pins the API's error contract: every known error, also when
// wrapped by a lower layer, maps to a fixed status and client message.
func TestError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err        error
		wantStatus int
		wantMsg    string
	}{
		{httputil.ErrInvalidJSON, http.StatusUnprocessableEntity, "invalid JSON body"},
		{httputil.ErrInvalidQuery, http.StatusUnprocessableEntity, "invalid query parameters"},
		{httputil.ErrInvalidPath, http.StatusUnprocessableEntity, "invalid path parameters"},
		{account.ErrAccountNotFound, http.StatusNotFound, "account not found"},
		{account.ErrInvalidCurrency, http.StatusUnprocessableEntity, "invalid currency code"},
		{account.ErrInvalidType, http.StatusUnprocessableEntity, "invalid account type"},
		{account.ErrAccountExists, http.StatusConflict, "account already exists"},
		{account.ErrDiscrepancyFound, http.StatusConflict, "account balance discrepancy detected"},
		{account.ErrInvalidDateRange, http.StatusUnprocessableEntity, "start_date must not be after end_date"},
		{account.ErrSnapshotExists, http.StatusConflict, "no new entries since the last balance snapshot"},
		{account.ErrSnapshotNotFound, http.StatusNotFound, "no balance snapshot exists for this account"},
		{transaction.ErrDuplicateReference, http.StatusUnprocessableEntity, "duplicate reference"},
		{transaction.ErrTransactionNotFound, http.StatusNotFound, "transaction not found"},
		{transaction.ErrInvalidCurrency, http.StatusUnprocessableEntity, "invalid currency code"},
		{
			transaction.ErrAtLeastTwoEntries,
			http.StatusUnprocessableEntity,
			"transaction must contain at least two entries",
		},
		{
			transaction.ErrUnbalancedTransaction,
			http.StatusUnprocessableEntity,
			"debits and credits must balance to zero",
		},
		{transaction.ErrInvalidEntryType, http.StatusUnprocessableEntity, "invalid entry type"},
		{transaction.ErrInvalidEntryAmount, http.StatusUnprocessableEntity, "entry amount must be greater than zero"},
		{transaction.ErrInvalidAccountID, http.StatusUnprocessableEntity, "entry account ID is required"},
		{
			transaction.ErrAmountOverflow,
			http.StatusUnprocessableEntity,
			"total debits or credits exceed the maximum amount",
		},
		{domain.ErrNotImplemented, http.StatusNotImplemented, "not implemented yet"},
		{errors.New("pq: connection refused"), http.StatusInternalServerError, "an unexpected error occurred"},
	}

	for _, tt := range tests {
		for _, wrapped := range []bool{false, true} {
			err := tt.err
			if wrapped {
				err = fmt.Errorf("layer: %w", tt.err)
			}

			t.Run(fmt.Sprintf("%v/wrapped=%v", tt.err, wrapped), func(t *testing.T) {
				t.Parallel()

				rec := httptest.NewRecorder()
				returned := httputil.Error(rec, err)

				if rec.Code != tt.wantStatus || returned != tt.wantStatus {
					t.Errorf("status written = %d, returned = %d, want %d", rec.Code, returned, tt.wantStatus)
				}

				var msg string
				if decodeErr := json.Unmarshal(rec.Body.Bytes(), &msg); decodeErr != nil || msg != tt.wantMsg {
					t.Errorf("body = %s, want JSON string %q", rec.Body, tt.wantMsg)
				}
			})
		}
	}
}

func TestError_ValidationErrors(t *testing.T) {
	t.Parallel()

	type request struct {
		OwnerID string `json:"owner_id" validate:"required"`
	}

	rec := httptest.NewRecorder()
	returned := httputil.Error(rec, validation.Struct(request{}))

	if rec.Code != http.StatusUnprocessableEntity || returned != http.StatusUnprocessableEntity {
		t.Fatalf("status written = %d, returned = %d, want 422", rec.Code, returned)
	}

	var got []validation.FieldError
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body %s is not a FieldError list: %v", rec.Body, err)
	}

	if len(got) != 1 || got[0] != (validation.FieldError{Field: "owner_id", Message: "required"}) {
		t.Errorf("field errors = %+v, want [{owner_id required}]", got)
	}
}

func TestError_Nil(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()

	if returned := httputil.Error(rec, nil); returned != 0 || rec.Body.Len() != 0 || rec.Code != http.StatusOK {
		t.Errorf("nil error returned %d and wrote status %d body %q; want 0 and nothing written",
			returned, rec.Code, rec.Body,
		)
	}
}
