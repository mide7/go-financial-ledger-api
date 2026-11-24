package account

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/mide7/go-financial-ledger-api/internal/domain"
	"github.com/mide7/go-financial-ledger-api/internal/domain/currency"
	"github.com/mide7/go-financial-ledger-api/internal/pagination"
)

// Page size bounds applied when a listing request omits or exceeds them.
const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

// ErrInvalidDateRange is returned when a statement's start date is after its end date.
var ErrInvalidDateRange = errors.New("start date must not be after end date")

// Service is the account use-case API consumed by the transport layer.
type Service interface {
	CreateAccount(ctx context.Context, params CreateAccountParams) (*Account, error)
	GetAccountDetails(ctx context.Context, id uuid.UUID) (*Account, error)
	ListAccounts(ctx context.Context, params ListAccountsParams) (pagination.OffsetPaginatedResult[Account], error)
	GetAccountBalance(ctx context.Context, id uuid.UUID) (*GetAccountBalanceResult, error)
	GetAccountStatementSummary(ctx context.Context, params GetStatementSummaryParams) (*StatementSummary, error)
	GetAccountStatementEntries(
		ctx context.Context,
		params GetStatementEntriesParams,
	) (pagination.CursorPaginatedResult[StatementEntry], error)
	ReconcileAccount(ctx context.Context, id uuid.UUID) (*ReconciliationResult, error)
	CreateAccountSnapshot(ctx context.Context, params CreateAccountSnapshotParams) (*BalanceSnapshot, error)
	GetLatestAccountSnapshot(ctx context.Context, id uuid.UUID) (*BalanceSnapshot, error)
	ChangeAccountStatus(ctx context.Context, params StatusChangeParams) (*StatusHistory, error)
	ListAccountStatusHistory(
		ctx context.Context,
		params ListAccountStatusHistoryParams,
	) (pagination.CursorPaginatedResult[StatusHistory], error)
	ListUnresolvedDiscrepancies(
		ctx context.Context,
		params ListDiscrepanciesParams,
	) (pagination.CursorPaginatedResult[DiscrepancyRecord], error)
	ResolveDiscrepancy(ctx context.Context, id uuid.UUID, notes string) error
	ReconcileAllAccounts(ctx context.Context) (*ReconciliationRun, error)
}

type accountService struct {
	repo             Repository
	currencyRegistry currency.Registry
	log              *slog.Logger
}

// NewAccountService returns a Service backed by repo and currencyRegistry that
// logs through log.
func NewAccountService(repo Repository, currencyRegistry currency.Registry, log *slog.Logger) Service {
	return &accountService{
		repo:             repo,
		currencyRegistry: currencyRegistry,
		log:              log,
	}
}

func (s *accountService) CreateAccount(ctx context.Context, params CreateAccountParams) (*Account, error) {
	if !Type(params.Type).IsValid() {
		return nil, ErrInvalidType
	}

	if _, active := s.currencyRegistry.Get(currency.Code(params.Currency)); !active {
		return nil, ErrInvalidCurrency
	}

	return s.repo.Create(ctx, params)
}

func (s *accountService) GetAccountDetails(ctx context.Context, id uuid.UUID) (*Account, error) {
	return s.repo.GetByID(ctx, id)
}

// ListAccounts returns one offset page of accounts with the total that match.
// Page defaults to 1 and Limit is clamped to 1..MaxPageLimit. An unknown Type
// is ErrInvalidType; an unknown SortBy or OrderBy is ErrInvalidSort.
func (s *accountService) ListAccounts(
	ctx context.Context,
	params ListAccountsParams,
) (pagination.OffsetPaginatedResult[Account], error) {
	var none pagination.OffsetPaginatedResult[Account]

	if params.Type != "" && !Type(params.Type).IsValid() {
		return none, ErrInvalidType
	}

	if err := normaliseSort(&params); err != nil {
		return none, err
	}

	params.Page = max(params.Page, 1)
	params.Limit = clampLimit(params.Limit)

	accounts, err := s.repo.List(ctx, params)
	if err != nil {
		return none, err
	}

	// A second query, so under concurrent writes the total can differ slightly
	// from what the pages add up to.
	total, err := s.repo.Count(ctx, params)
	if err != nil {
		return none, err
	}

	return pagination.NewOffsetPaginatedResult(accounts, params.Page, params.Limit, total), nil
}

// normaliseSort applies the sort defaults and rejects unknown values, so the
// query's ORDER BY always gets a field it knows with a direction.
func normaliseSort(params *ListAccountsParams) error {
	if params.SortBy == "" {
		params.SortBy = SortByCreatedAt
	}

	if params.OrderBy == "" {
		params.OrderBy = SortOrderDesc
	}

	switch params.SortBy {
	case SortByCreatedAt, SortByType, SortByCurrency:
	default:
		return ErrInvalidSort
	}

	switch params.OrderBy {
	case SortOrderAsc, SortOrderDesc:
	default:
		return ErrInvalidSort
	}

	return nil
}

func (s *accountService) GetAccountBalance(ctx context.Context, id uuid.UUID) (*GetAccountBalanceResult, error) {
	return s.repo.GetBalance(ctx, id)
}

func (s *accountService) GetAccountStatementSummary(
	ctx context.Context,
	params GetStatementSummaryParams,
) (*StatementSummary, error) {
	if params.StartDate.After(params.EndDate) {
		return nil, ErrInvalidDateRange
	}

	return s.repo.GetStatementSummary(ctx, params)
}

func (s *accountService) GetAccountStatementEntries(
	ctx context.Context,
	params GetStatementEntriesParams,
) (pagination.CursorPaginatedResult[StatementEntry], error) {
	if params.StartDate.After(params.EndDate) {
		return pagination.CursorPaginatedResult[StatementEntry]{}, ErrInvalidDateRange
	}

	limit := clampLimit(params.Limit)
	params.Limit = pagination.FetchLimit(limit)

	entries, err := s.repo.ListStatementEntries(ctx, params)
	if err != nil {
		return pagination.CursorPaginatedResult[StatementEntry]{}, err
	}

	return pagination.NewCursorPaginatedResult(entries, limit, func(e StatementEntry) pagination.Cursor {
		return pagination.Cursor{ID: e.ID}
	}), nil
}

// ReconcileAccount recomputes the balance from the ledger and compares it with the
// latest snapshot. A mismatch is persisted as a DiscrepancyRecord and still
// returned as a result rather than an error, so callers can inspect the figures.
func (s *accountService) ReconcileAccount(ctx context.Context, id uuid.UUID) (*ReconciliationResult, error) {
	result, err := s.repo.Reconcile(ctx, id)
	if err != nil {
		return nil, err
	}

	if result.Status != ReconciliationStatusDiscrepancy {
		return result, nil
	}

	if _, err := s.repo.CreateDiscrepancyRecord(ctx, CreateDiscrepancyRecordParams{
		AccountID:            result.AccountID,
		RawLedgerSum:         result.RawLedgerSum,
		SnapshotBalance:      result.SnapshotBalance,
		CalculatedTotal:      result.CalculatedTotal,
		DiscrepancyAmount:    result.DiscrepancyAmount,
		LastSnapshotSequence: result.LastSnapshotSequence,
		CurrentMaxSequence:   result.CurrentMaxSequence,
	}); err != nil {
		return nil, fmt.Errorf("record discrepancy: %w", err)
	}

	s.log.WarnContext(ctx, "balance discrepancy detected",
		"account_id", result.AccountID,
		"discrepancy_amount", result.DiscrepancyAmount,
	)

	return result, nil
}

func (s *accountService) CreateAccountSnapshot(
	ctx context.Context,
	params CreateAccountSnapshotParams,
) (*BalanceSnapshot, error) {
	if _, err := s.repo.GetByID(ctx, params.ID); err != nil {
		return nil, err
	}

	snapshot, err := s.repo.CreateBalanceSnapshot(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "balance snapshot created",
		"account_id", params.ID,
		"last_entry_sequence", snapshot.LastEntrySequence,
		"reason", params.Reason,
	)

	return snapshot, nil
}

func (s *accountService) GetLatestAccountSnapshot(ctx context.Context, id uuid.UUID) (*BalanceSnapshot, error) {
	return s.repo.GetLatestSnapshot(ctx, id)
}

// ChangeAccountStatus moves the account to params.NewStatus and records the
// change in its status history. When ExpectedStatus is set, the change only
// applies if the account is still in that status (ErrStatusConflict otherwise).
func (s *accountService) ChangeAccountStatus(ctx context.Context, params StatusChangeParams) (*StatusHistory, error) {
	if !params.NewStatus.IsValid() || (params.ExpectedStatus != nil && !params.ExpectedStatus.IsValid()) {
		return nil, ErrInvalidStatus
	}

	if !params.ActorType.IsValid() {
		return nil, ErrInvalidActorType
	}

	history, err := s.repo.ChangeAccountStatus(ctx, params)
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "account status changed",
		"account_id", params.AccountID,
		"new_status", params.NewStatus,
		"actor_type", params.ActorType,
		"actor_id", params.ActorID,
	)

	return history, nil
}

func (s *accountService) ListAccountStatusHistory(
	ctx context.Context,
	params ListAccountStatusHistoryParams,
) (pagination.CursorPaginatedResult[StatusHistory], error) {
	limit := clampLimit(params.Limit)
	params.Limit = pagination.FetchLimit(limit)

	history, err := s.repo.ListAccountStatusHistory(ctx, params)
	if err != nil {
		return pagination.CursorPaginatedResult[StatusHistory]{}, err
	}

	return pagination.NewCursorPaginatedResult(history, limit, func(h StatusHistory) pagination.Cursor {
		return pagination.Cursor{ID: h.ID, CreatedAt: h.CreatedAt}
	}), nil
}

func (s *accountService) ListUnresolvedDiscrepancies(
	ctx context.Context,
	params ListDiscrepanciesParams,
) (pagination.CursorPaginatedResult[DiscrepancyRecord], error) {
	limit := clampLimit(params.Limit)
	params.Limit = pagination.FetchLimit(limit)

	records, err := s.repo.ListUnresolvedDiscrepancies(ctx, params)
	if err != nil {
		return pagination.CursorPaginatedResult[DiscrepancyRecord]{}, err
	}

	return pagination.NewCursorPaginatedResult(records, limit, func(d DiscrepancyRecord) pagination.Cursor {
		return pagination.Cursor{ID: d.ID}
	}), nil
}

func (s *accountService) ResolveDiscrepancy(ctx context.Context, id uuid.UUID, notes string) error {
	if err := s.repo.ResolveDiscrepancy(ctx, id, notes); err != nil {
		return err
	}

	s.log.InfoContext(ctx, "discrepancy resolved", "discrepancy_id", id)

	return nil
}

// ReconcileAllAccounts reconciles every account in pages of ids, recording
// discrepancies as ReconcileAccount does.
//
// TODO(later-reconcile-all): page through s.repo.ListAccountIDs and call
// s.ReconcileAccount for each; see docs/todo.md.
func (s *accountService) ReconcileAllAccounts(_ context.Context) (*ReconciliationRun, error) {
	return nil, domain.ErrNotImplemented
}

func clampLimit[T int | int32](limit T) T {
	if limit < 1 {
		return DefaultPageLimit
	}

	return min(limit, MaxPageLimit)
}
