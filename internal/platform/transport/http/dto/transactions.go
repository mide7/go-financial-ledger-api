package dto

type CreateTransactionDTO struct {
	Reference   string `json:"reference" validate:"required,min=1,max=128"`
	Type        string `json:"type" validate:"required,min=1,max=64"`
	Status      string `json:"status" validate:"required,oneof=POSTED REVERSED FAILED"`
	Description string `json:"description" validate:"required,min=1,max=256"`
	Metadata    string `json:"metadata" validate:"omitempty"`
}

type ListTransactionsDTO struct {
	Page   int    `json:"page" validate:"omitempty,gte=1"`
	Limit  int    `json:"limit" validate:"omitempty,gte=1,lte=100"`
	Type   string `json:"type" validate:"omitempty,min=1,max=64"`
	Status string `json:"status" validate:"omitempty,oneof=POSTED REVERSED FAILED"`
}
