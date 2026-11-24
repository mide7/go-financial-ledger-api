package repository

import (
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const pgErrUniqueViolation = "23505"

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
