package repository

import (
	"testing"

	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

// TestCurrencyRegistry_Get covers lookups against an already-loaded cache.
// Loading and LISTEN/NOTIFY refresh need a database and are not covered here.
func TestCurrencyRegistry_Get(t *testing.T) {
	t.Parallel()

	r := &CurrencyRegistry{currencies: map[currency.Code]currency.Currency{
		currency.NGN: {Code: currency.NGN, Exponent: 2, IsActive: true},
		currency.EUR: {Code: currency.EUR, Exponent: 2, IsActive: false},
	}}

	tests := []struct {
		name       string
		code       currency.Code
		wantActive bool
	}{
		{name: "active currency", code: currency.NGN, wantActive: true},
		{name: "inactive currency is not usable", code: currency.EUR, wantActive: false},
		{name: "unknown currency", code: currency.USD, wantActive: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, active := r.Get(tt.code)
			if active != tt.wantActive {
				t.Fatalf("Get(%s) active = %v, want %v", tt.code, active, tt.wantActive)
			}

			if tt.wantActive && got.Code != tt.code {
				t.Errorf("Get(%s) = %+v", tt.code, got)
			}
		})
	}
}

func TestCurrencyRegistry_List(t *testing.T) {
	t.Parallel()

	r := &CurrencyRegistry{currencies: map[currency.Code]currency.Currency{
		currency.USD: {Code: currency.USD, IsActive: true},
		currency.NGN: {Code: currency.NGN, IsActive: true},
		currency.EUR: {Code: currency.EUR, IsActive: false},
	}}

	got := r.List()

	if len(got) != 2 || got[0].Code != currency.NGN || got[1].Code != currency.USD {
		t.Errorf("List() = %+v, want active NGN then USD", got)
	}
}
