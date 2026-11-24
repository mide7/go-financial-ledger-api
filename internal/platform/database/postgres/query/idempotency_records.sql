-- name: InsertIdempotencyRecord :one
-- Claims key for a new request. Returns no row when the key already exists.
INSERT INTO idempotency_records (key, request_hash, expires_at)
VALUES (
        sqlc.arg('key'),
        sqlc.arg('request_hash'),
        sqlc.arg('expires_at')
    ) ON CONFLICT (key) DO NOTHING
RETURNING *;

-- name: ReclaimIdempotencyRecord :one
-- Takes over an existing key for a new attempt when its record has expired
-- (any request) or FAILED for the same request. Returns no row otherwise.
UPDATE idempotency_records
SET request_hash = sqlc.arg('request_hash'),
    status = 'IN_PROGRESS',
    status_code = NULL,
    response_body = NULL,
    expires_at = sqlc.arg('expires_at')
WHERE key = sqlc.arg('key')
    AND (
        expires_at < NOW()
        OR (
            status = 'FAILED'
            AND request_hash = sqlc.arg('request_hash')
        )
    )
RETURNING *;

-- name: GetIdempotencyRecord :one
SELECT *
FROM idempotency_records
WHERE key = sqlc.arg('key');

-- name: CompleteIdempotencyRecord :execrows
-- Stores the response for a claimed key. Status is COMPLETED or FAILED.
UPDATE idempotency_records
SET status = sqlc.arg('status'),
    status_code = sqlc.arg('status_code'),
    response_body = sqlc.arg('response_body')
WHERE key = sqlc.arg('key')
    AND status = 'IN_PROGRESS';

-- name: DeleteExpiredIdempotencyRecords :execrows
DELETE FROM idempotency_records
WHERE expires_at < NOW();
