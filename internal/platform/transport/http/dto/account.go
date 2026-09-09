package dto

type CreateAccountDTO struct {
	OwnerId  string `json:"owner_id" validate:"required,min=1,max=64"`
	Type     string `json:"type" validate:"required,oneof=USER_WALLET PLATFORM_ESCROW_LIABILITY PLATFORM_FEE_REVENUEMERCHANT_PAYABLE"`
	Currency string `json:"currency" validate:"required,len=3,uppercase"`
}

type ListAccountsDTO struct {
	Page     int    `json:"page" validate:"omitempty,gte=1"`
	Limit    int    `json:"limit" validate:"omitempty,gte=1,lte=100"`
	Currency string `json:"currency" validate:"omitempty,len=3,uppercase"`
	Type     string `json:"type" validate:"omitempty,oneof=USER_WALLET PLATFORM_ESCROW_LIABILITY PLATFORM_FEE_REVENUEMERCHANT_PAYABLE"`
}
