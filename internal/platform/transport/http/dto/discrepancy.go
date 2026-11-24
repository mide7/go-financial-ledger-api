package dto

import "github.com/google/uuid"

// ResolveDiscrepancyDTO holds the parameters of POST /discrepancies/{id}/resolve.
type ResolveDiscrepancyDTO struct {
	ID    uuid.UUID `form:"id" validate:"required"`
	Notes string    `json:"notes" validate:"required,min=1,max=1024"`
}
