package repository

import (
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/mide7/go-financial-ledger-api/internal/domain/transaction"
)

const pgErrUniqueViolation = "23505"

// SQLSTATEs raised by the trg_check_transaction_integrity trigger.
const (
	pgErrCurrencyMismatch = "LD001"
	pgErrUnbalanced       = "LD002"
)

// uniqueOneReversal is the partial unique index allowing one REVERSAL per parent.
const uniqueOneReversal = "uq_transactions_one_reversal"

var errOutOfRange = errors.New("value out of int32 range")

// notFound replaces a "no rows" error with the domain's not-found error and
// passes any other error through unchanged.
func notFound(err, domainErr error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domainErr
	}

	return err
}

func isUniqueViolation(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == pgErrUniqueViolation
	}

	return false
}

// mapIntegrityError turns the integrity trigger's exceptions into domain
// errors and passes any other error through.
func mapIntegrityError(err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return err
	}

	switch pgErr.Code {
	case pgErrCurrencyMismatch:
		return fmt.Errorf("%w: %s", transaction.ErrCurrencyMismatch, pgErr.Message)
	case pgErrUnbalanced:
		return fmt.Errorf("%w: %s", transaction.ErrUnbalancedTransaction, pgErr.Message)
	default:
		return err
	}
}

// isUniqueViolationOf reports whether err violates the named unique constraint or index.
func isUniqueViolationOf(err error, constraint string) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == pgErrUniqueViolation && pgErr.ConstraintName == constraint
	}

	return false
}

// pageOffset returns the row offset of a 1-based page. Pages below 1 are
// treated as the first page.
func pageOffset(page, limit int) (int32, error) {
	return toInt32((max(page, 1) - 1) * limit)
}

// toInt32 converts n for use as an int32 query parameter, rejecting values
// that would otherwise wrap around silently.
func toInt32(n int) (int32, error) {
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, fmt.Errorf("%w: %d", errOutOfRange, n)
	}

	return int32(n), nil
}
