-- name: CreateTransaction :one
INSERT INTO transactions (
        reference,
        type,
        currency,
        parent_transaction_id,
        description,
        metadata,
        transaction_date
    )
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;
-- name: CreateEntries :copyfrom
INSERT INTO entries (
        transaction_id,
        account_id,
        type,
        amount
    )
VALUES ($1, $2, $3, $4);
-- name: GetTransactionByID :one
SELECT *
FROM transactions
WHERE id = sqlc.arg('id');

-- name: ListTransactions :many
-- Newest first. The cursor is the id of the last row on the previous page;
-- ids are UUIDv7, so id order is creation order. Filters are optional.
SELECT t.*
FROM transactions t
WHERE (
        sqlc.narg('account_id')::uuid IS NULL
        OR EXISTS (
            SELECT 1
            FROM entries e
            WHERE e.transaction_id = t.id
                AND e.account_id = sqlc.narg('account_id')::uuid
        )
    )
    AND (
        sqlc.narg('reference')::text IS NULL
        OR t.reference = sqlc.narg('reference')::text
    )
    AND (
        sqlc.narg('type')::text IS NULL
        OR t.type = sqlc.narg('type')::text
    )
    AND (
        sqlc.narg('currency')::text IS NULL
        OR t.currency = sqlc.narg('currency')::text
    )
    AND (
        sqlc.narg('start_date')::timestamptz IS NULL
        OR t.created_at >= sqlc.narg('start_date')::timestamptz
    )
    AND (
        sqlc.narg('end_date')::timestamptz IS NULL
        OR t.created_at < sqlc.narg('end_date')::timestamptz
    )
    AND (
        sqlc.narg('cursor')::uuid IS NULL
        OR t.id < sqlc.narg('cursor')::uuid
    )
ORDER BY t.id DESC
LIMIT sqlc.arg('limit')::int;

-- name: GetReversalOf :one
-- The reversal posted for a transaction, if any.
SELECT *
FROM transactions
WHERE parent_transaction_id = sqlc.arg('transaction_id')
    AND type = 'REVERSAL'
LIMIT 1;
