package transaction

import (
	"context"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
)

// Page size bounds applied when a listing request omits or exceeds them.
const (
	DefaultPageLimit int32 = 20
	MaxPageLimit     int32 = 100
)

// Service is the transaction use-case API consumed by the transport layer.
type Service interface {
	CreateTransaction(ctx context.Context, params CreateTransactionParams) (*Transaction, error)
	GetTransactionDetails(ctx context.Context, id uuid.UUID) (*Transaction, error)
	ListTransactions(ctx context.Context, params ListTransactionsParams) ([]Transaction, error)
	ReverseTransaction(ctx context.Context, params ReverseTransactionParams) (*Transaction, error)
	CreateConversion(ctx context.Context, params CreateConversionParams) (*Conversion, error)
}

type transactionService struct {
	repo             Repository
	currencyRegistry currency.Registry
}

// NewTransactionService returns a Service backed by repo and currencyRegistry.
func NewTransactionService(repo Repository, currencyRegistry currency.Registry) Service {
	return &transactionService{
		repo:             repo,
		currencyRegistry: currencyRegistry,
	}
}

func (s *transactionService) CreateTransaction(
	ctx context.Context,
	params CreateTransactionParams,
) (*Transaction, error) {
	if params.Type == TypeReversal {
		return nil, ErrReservedType
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	if _, active := s.currencyRegistry.Get(params.Currency); !active {
		return nil, ErrInvalidCurrency
	}

	params.Metadata = params.metadataOrEmpty()

	return s.repo.Create(ctx, params)
}

// GetTransactionDetails returns the transaction and its entries.
func (s *transactionService) GetTransactionDetails(ctx context.Context, id uuid.UUID) (*Transaction, error) {
	return s.repo.GetByID(ctx, id)
}

// ListTransactions returns transaction headers, newest first. A limit outside
// 1..MaxPageLimit becomes DefaultPageLimit or MaxPageLimit. When both dates are
// set, StartDate must be before EndDate (the range is half-open).
func (s *transactionService) ListTransactions(
	ctx context.Context,
	params ListTransactionsParams,
) ([]Transaction, error) {
	if !params.StartDate.IsZero() && !params.EndDate.IsZero() && !params.StartDate.Before(params.EndDate) {
		return nil, ErrInvalidDateRange
	}

	params.Limit = clampLimit(params.Limit)

	return s.repo.List(ctx, params)
}

// ReverseTransaction posts a REVERSAL transaction that mirrors the entries of
// transaction params.ID, debits for credits. A transaction can be reversed
// once, and a reversal can't itself be reversed.
func (s *transactionService) ReverseTransaction(
	ctx context.Context,
	params ReverseTransactionParams,
) (*Transaction, error) {
	return s.repo.Reverse(ctx, params)
}

// TODO(later-fx): reject equal source and target accounts, then post both legs
// through PLATFORM_FX_SUSPENSE; see docs/todo.md.
func (s *transactionService) CreateConversion(_ context.Context, _ CreateConversionParams) (*Conversion, error) {
	return nil, domain.ErrNotImplemented
}

func clampLimit(limit int32) int32 {
	if limit < 1 {
		return DefaultPageLimit
	}

	return min(limit, MaxPageLimit)
}
