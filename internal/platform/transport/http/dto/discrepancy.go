package dto

import (
	"github.com/google/uuid"

	"github.com/mide7/go-financial-ledger-api/internal/domain/account"
)

// ListDiscrepanciesDTO holds the query parameters of GET /discrepancies.
// cursor is the id of the last record on the previous page.
type ListDiscrepanciesDTO struct {
	Cursor uuid.NullUUID `form:"cursor"`
	Limit  int32         `form:"limit" validate:"omitempty,gte=1,lte=100"`
}

// ToDomainParams converts the request to service parameters.
func (l ListDiscrepanciesDTO) ToDomainParams() account.ListDiscrepanciesParams {
	return account.ListDiscrepanciesParams{Cursor: l.Cursor, Limit: l.Limit}
}

// ResolveDiscrepancyDTO holds the parameters of POST /discrepancies/{id}/resolve.
type ResolveDiscrepancyDTO struct {
	ID    uuid.UUID `form:"id" validate:"required"`
	Notes string    `json:"notes" validate:"required,min=1,max=1024"`
}
