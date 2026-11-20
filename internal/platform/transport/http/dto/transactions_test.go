package dto_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/platform/transport/http/dto"
)

func TestCreateTransactionDTO_ToDomainParams(t *testing.T) {
	t.Parallel()

	accountID := uuid.New()
	parentID := uuid.New()
	parent := parentID.String()
	empty := ""
	bad := "not-a-uuid"

	base := func() dto.CreateTransactionDTO {
		return dto.CreateTransactionDTO{
			Reference: "ref-1",
			Type:      "TRANSFER",
			Currency:  "NGN",
			Entries: []dto.EntryDTO{
				{AccountID: accountID.String(), Type: "DEBIT", Amount: 100},
				{AccountID: accountID.String(), Type: "CREDIT", Amount: 100},
			},
		}
	}

	tests := []struct {
		name       string
		mutate     func(*dto.CreateTransactionDTO)
		wantParent uuid.NullUUID
		wantErr    bool
	}{
		{name: "no parent", mutate: func(*dto.CreateTransactionDTO) {}},
		{
			name:   "empty parent is no parent",
			mutate: func(d *dto.CreateTransactionDTO) { d.ParentTransactionID = &empty },
		},
		{
			name:       "parent is parsed",
			mutate:     func(d *dto.CreateTransactionDTO) { d.ParentTransactionID = &parent },
			wantParent: uuid.NullUUID{UUID: parentID, Valid: true},
		},
		{
			name:    "malformed parent",
			mutate:  func(d *dto.CreateTransactionDTO) { d.ParentTransactionID = &bad },
			wantErr: true,
		},
		{
			name:    "malformed entry account",
			mutate:  func(d *dto.CreateTransactionDTO) { d.Entries[1].AccountID = bad },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := base()
			tt.mutate(&in)

			got, err := in.ToDomainParams()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ToDomainParams() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if got.Reference != "ref-1" || got.Currency != currency.NGN || got.ParentTransactionID != tt.wantParent {
				t.Errorf("params = %+v", got)
			}

			if len(got.Entries) != 2 || got.Entries[0].AccountID != accountID ||
				got.Entries[0].Type != domain.EntryTypeDebit || got.Entries[1].Type != domain.EntryTypeCredit ||
				got.Entries[1].Amount != 100 {
				t.Errorf("entries = %+v", got.Entries)
			}
		})
	}
}
