// Package idempotency defines the store behind the Idempotency-Key header:
// the first request with a key runs and its response is saved, and a retry
// with the same key and request gets the saved response instead of running
// again. The HTTP middleware lives in transport/http/middleware.
package idempotency

import (
	"context"
	"errors"
	"time"
)

// Errors returned by a Store or reported by the middleware.
var (
	// ErrKeyReused means the key was first used with a different request.
	ErrKeyReused = errors.New("idempotency key reused with a different request")
	// ErrInProgress means the first request with this key has not finished.
	ErrInProgress = errors.New("a request with this idempotency key is in progress")
	// ErrNotClaimed means Complete or Fail found no IN_PROGRESS record for the
	// key, because it expired and was reclaimed or was never claimed.
	ErrNotClaimed = errors.New("idempotency key is not claimed")
)

// Record statuses, matching the idempotency_records CHECK constraint.
const (
	StatusInProgress = "IN_PROGRESS"
	StatusCompleted  = "COMPLETED"
	StatusFailed     = "FAILED"
)

// Record is a stored request and, once finished, its response.
type Record struct {
	Key          string
	RequestHash  string
	Status       string
	StatusCode   int    // 0 until finished
	ResponseBody []byte // a JSON document; nil until completed
	ExpiresAt    time.Time
}

// Store persists idempotency records. Implemented on Postgres in
// internal/platform/database/postgres/repository.
type Store interface {
	// Claim reserves key for a request whose fingerprint is requestHash, for
	// ttl. It also takes over a key whose record has expired, or FAILED for
	// the same requestHash. When claimed is false, existing is the stored
	// record for the caller to replay or reject.
	Claim(ctx context.Context, key, requestHash string, ttl time.Duration) (existing *Record, claimed bool, err error)
	// Complete saves the response for a claimed key; body must be valid JSON.
	Complete(ctx context.Context, key string, statusCode int, body []byte) error
	// Fail marks a claimed key FAILED, so a retry of the same request runs again.
	Fail(ctx context.Context, key string, statusCode int) error
	// DeleteExpired removes records past their expiry and reports how many.
	DeleteExpired(ctx context.Context) (int64, error)
}
