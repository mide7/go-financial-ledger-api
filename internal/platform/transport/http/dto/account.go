package dto

type CreateAccountDTO struct {
	OwnerId  string `json:"owner_id" validate:"required,min=1,max=64"`
	Type     string `json:"type" validate:"required,oneof=USER_WALLET PLATFORM_ESCROW_LIABILITY PLATFORM_FEE_REVENUEMERCHANT_PAYABLE"`
	Currency string `json:"currency" validate:"required,len=3,uppercase"`
}

type ListAccountsDTO struct {
	Page     int    `form:"page" validate:"omitempty,gte=1"`
	Limit    int    `form:"limit" validate:"omitempty,gte=1,lte=100"`
	Currency string `form:"currency" validate:"omitempty,len=3,uppercase"`
	Type     string `form:"type" validate:"omitempty,oneof=USER_WALLET PLATFORM_ESCROW_LIABILITY PLATFORM_FEE_REVENUEMERCHANT_PAYABLE"`
}
