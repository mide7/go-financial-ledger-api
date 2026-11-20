package validation

import (
	"errors"
	"slices"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestGet(t *testing.T) {
	t.Parallel()

	first, second := Get(), Get()
	if first != second {
		t.Fatal("Get() returned different validators; want one shared instance")
	}
}

func TestStruct(t *testing.T) {
	t.Parallel()

	type request struct {
		OwnerID  string `json:"owner_id" validate:"required"`
		Page     int    `form:"page" validate:"omitempty,gte=1"`
		Internal string `json:"-" validate:"required"`
		Untagged string `validate:"required"`
	}

	valid := request{OwnerID: "u1", Page: 1, Internal: "x", Untagged: "x"}

	tests := []struct {
		name       string
		input      request
		wantFields []string
	}{
		{name: "valid", input: valid},
		{
			// Field names follow the client-facing tag: json first, then form,
			// then the Go name when a tag is missing or "-".
			name:       "every field invalid",
			input:      request{Page: -1},
			wantFields: []string{"owner_id", "page", "Internal", "Untagged"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := Struct(tt.input)
			if len(tt.wantFields) == 0 {
				if err != nil {
					t.Fatalf("Struct() error = %v, want nil", err)
				}

				return
			}

			var ve validator.ValidationErrors
			if !errors.As(err, &ve) {
				t.Fatalf("Struct() error = %v, want validator.ValidationErrors", err)
			}

			got := make([]string, len(ve))
			for i, fe := range ve {
				got[i] = fe.Field()
			}

			if !slices.Equal(got, tt.wantFields) {
				t.Errorf("failed fields = %v, want %v", got, tt.wantFields)
			}
		})
	}
}
