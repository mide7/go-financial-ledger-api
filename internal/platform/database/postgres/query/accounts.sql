-- name: CreateAccount :one
INSERT INTO accounts (owner_id, type, currency)
VALUES ($1, $2, $3)
RETURNING *;
-- name: GetAccountBalance :one
WITH latest_snapshot AS (
    SELECT balance,
        last_entry_sequence
    FROM balance_snapshots
    WHERE account_id = '123e4567-e89b-12d3-a456-426614174000'
    ORDER BY last_entry_sequence DESC
    LIMIT 1
)
SELECT COALESCE(s.balance, 0) + COALESCE(
        SUM(
            CASE
                WHEN e.type = 'CREDIT' THEN e.amount
                WHEN e.type = 'DEBIT' THEN - e.amount
                ELSE 0
            END
        ),
        0
    ) AS current_balance
FROM (
        SELECT balance,
            last_entry_sequence
        FROM latest_snapshot
        UNION ALL
        -- Fallback zero-state if no snapshot exists yet for a new account
        SELECT 0 AS balance,
            0 AS last_entry_sequence
        LIMIT 1
    ) s
    LEFT JOIN entries e ON e.account_id = '123e4567-e89b-12d3-a456-426614174000'
    AND e.entry_sequence > s.last_entry_sequence;