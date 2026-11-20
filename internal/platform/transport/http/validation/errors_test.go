package validation

import (
	"errors"
	"testing"
	"time"
)

func TestFormatValidationErrors(t *testing.T) {
	t.Parallel()

	// One field per supported tag, each given a value that fails it.
	type request struct {
		Required  string    `json:"required" validate:"required"`
		UUID      string    `json:"uuid" validate:"uuid"`
		GT        int       `json:"gt" validate:"gt=0"`
		GTE       int       `json:"gte" validate:"gte=1"`
		LT        int       `json:"lt" validate:"lt=10"`
		LTE       int       `json:"lte" validate:"lte=10"`
		Len       string    `json:"len" validate:"len=3"`
		OneOf     string    `json:"oneof" validate:"oneof=DEBIT CREDIT"`
		Min       string    `json:"min" validate:"min=2"`
		Max       string    `json:"max" validate:"max=2"`
		Uppercase string    `json:"uppercase" validate:"uppercase"`
		Lowercase string    `json:"lowercase" validate:"lowercase"`
		Alphanum  string    `json:"alphanum" validate:"alphanum"`
		IsDefault time.Time `json:"isdefault" validate:"isdefault"`
		Unmapped  string    `json:"unmapped" validate:"email"`
		NoTag     string    `validate:"required"`
	}

	err := Struct(request{
		UUID:      "not-a-uuid",
		GT:        0,
		GTE:       0,
		LT:        10,
		LTE:       11,
		Len:       "ab",
		OneOf:     "SIDEWAYS",
		Min:       "a",
		Max:       "abc",
		Uppercase: "abc",
		Lowercase: "ABC",
		Alphanum:  "a-b",
		IsDefault: time.Unix(1, 0),
		Unmapped:  "not-an-email",
	})

	want := map[string]string{
		"required":  "required",
		"uuid":      "must be a valid UUID",
		"gt":        "must be greater than 0",
		"gte":       "must be greater than or equal to 1",
		"lt":        "must be less than 10",
		"lte":       "must be less than or equal to 10",
		"len":       "must be exactly 3 characters",
		"oneof":     "must be one of: [DEBIT CREDIT]",
		"min":       "must be at least 2",
		"max":       "must be at most 2",
		"uppercase": "must be uppercase",
		"lowercase": "must be lowercase",
		"alphanum":  "must be alphanumeric",
		"isdefault": "is not supported yet; omit this field",
		"unmapped":  "is invalid",
		"notag":     "required", // Go field name, lowercased
	}

	got := FormatValidationErrors(err)
	if len(got) != len(want) {
		t.Fatalf("got %d field errors, want %d: %+v", len(got), len(want), got)
	}

	for _, fe := range got {
		wantMsg, ok := want[fe.Field]
		if !ok {
			t.Errorf("unexpected field %q (message %q)", fe.Field, fe.Message)
			continue
		}

		if fe.Message != wantMsg {
			t.Errorf("field %q: message = %q, want %q", fe.Field, fe.Message, wantMsg)
		}
	}
}

func TestFormatValidationErrors_NonValidatorError(t *testing.T) {
	t.Parallel()

	got := FormatValidationErrors(errors.New("boom"))
	if len(got) != 1 || got[0].Field != "unknown" || got[0].Message != "boom" {
		t.Errorf("got %+v, want a single unknown/boom entry", got)
	}
}
