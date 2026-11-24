package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mide7/go-financial-ledger-api/internal/platform/database/postgres/sqlc"
	"github.com/mide7/go-financial-ledger-api/internal/platform/idempotency"
)

// claimAttempts bounds Claim's insert → reclaim → get loop. A second attempt is
// needed only when the record is deleted between the failed insert and the get.
const claimAttempts = 2

var errClaimRace = errors.New("idempotency record changed during claim")

type idempotencyStore struct {
	queries *sqlc.Queries
}

// NewIdempotencyStore returns an idempotency.Store backed by the
// idempotency_records table.
func NewIdempotencyStore(queries *sqlc.Queries) idempotency.Store {
	return &idempotencyStore{queries: queries}
}

// Claim inserts the key; if it exists, tries to reclaim it (expired, or FAILED
// for the same request); otherwise returns the stored record. Each step is a
// single statement, so two concurrent claims can't both win.
func (s *idempotencyStore) Claim(
	ctx context.Context,
	key, requestHash string,
	ttl time.Duration,
) (*idempotency.Record, bool, error) {
	for range claimAttempts {
		expiresAt := time.Now().Add(ttl)

		_, err := s.queries.InsertIdempotencyRecord(ctx, sqlc.InsertIdempotencyRecordParams{
			Key: key, RequestHash: requestHash, ExpiresAt: expiresAt,
		})
		if err == nil {
			return nil, true, nil
		}

		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, false, err
		}

		_, err = s.queries.ReclaimIdempotencyRecord(ctx, sqlc.ReclaimIdempotencyRecordParams{
			Key: key, RequestHash: requestHash, ExpiresAt: expiresAt,
		})
		if err == nil {
			return nil, true, nil
		}

		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, false, err
		}

		existing, err := s.queries.GetIdempotencyRecord(ctx, key)
		if errors.Is(err, pgx.ErrNoRows) {
			continue // purged since the insert; try again
		}

		if err != nil {
			return nil, false, err
		}

		return mapToIdempotencyRecord(&existing), false, nil
	}

	return nil, false, fmt.Errorf("%w: %s", errClaimRace, key)
}

func (s *idempotencyStore) Complete(ctx context.Context, key string, statusCode int, body []byte) error {
	return s.finish(ctx, key, idempotency.StatusCompleted, statusCode, body)
}

func (s *idempotencyStore) Fail(ctx context.Context, key string, statusCode int) error {
	return s.finish(ctx, key, idempotency.StatusFailed, statusCode, nil)
}

func (s *idempotencyStore) finish(ctx context.Context, key, status string, statusCode int, body []byte) error {
	if statusCode < 0 || statusCode > math.MaxInt32 {
		return fmt.Errorf("status code: %w: %d", errOutOfRange, statusCode)
	}

	n, err := s.queries.CompleteIdempotencyRecord(ctx, sqlc.CompleteIdempotencyRecordParams{
		Key:          key,
		Status:       status,
		StatusCode:   pgtype.Int4{Int32: int32(statusCode), Valid: true},
		ResponseBody: body,
	})
	if err != nil {
		return err
	}

	if n == 0 {
		return fmt.Errorf("%w: %s", idempotency.ErrNotClaimed, key)
	}

	return nil
}

func (s *idempotencyStore) DeleteExpired(ctx context.Context) (int64, error) {
	return s.queries.DeleteExpiredIdempotencyRecords(ctx)
}

func mapToIdempotencyRecord(r *sqlc.IdempotencyRecord) *idempotency.Record {
	return &idempotency.Record{
		Key:          r.Key,
		RequestHash:  r.RequestHash,
		Status:       r.Status,
		StatusCode:   int(r.StatusCode.Int32),
		ResponseBody: r.ResponseBody,
		ExpiresAt:    r.ExpiresAt,
	}
}
