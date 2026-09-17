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