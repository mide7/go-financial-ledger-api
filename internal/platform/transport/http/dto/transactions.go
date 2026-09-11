package dto

type CreateTransactionDTO struct {
	Reference   string `json:"reference" validate:"required,min=1,max=128"`
	Type        string `json:"type" validate:"required,min=1,max=64"`
	Status      string `json:"status" validate:"required,oneof=POSTED REVERSED FAILED"`
	Description string `json:"description" validate:"required,min=1,max=256"`
	Metadata    string `json:"metadata" validate:"omitempty"`
}

type GetTransactionDetailsDTO struct {
	ID string `form:"id" validate:"required,uuid"`
}

type ListTransactionsDTO struct {
	Page   int    `form:"page" validate:"omitempty,gte=1"`
	Limit  int    `form:"limit" validate:"omitempty,gte=1,lte=100"`
	Type   string `form:"type" validate:"omitempty,min=1,max=64"`
	Status string `form:"status" validate:"omitempty,oneof=POSTED REVERSED FAILED"`
}

type ReverseTransactionDTO struct {
	ID string `form:"id" validate:"required,uuid"`
}

type GetTransactionEntriesDTO struct {
	ID string `form:"id" validate:"required,uuid"`
}
