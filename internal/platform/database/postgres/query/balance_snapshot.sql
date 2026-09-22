-- name: CreateBalanceSnapshot :one
WITH current_balance AS (
    SELECT COALESCE(
            SUM(
                CASE
                    WHEN type = 'DEBIT' THEN amount
                    ELSE - amount
                END
            ),
            0
        )::bigint AS balance,
        COALESCE(MAX(entry_sequence), 0)::bigint AS last_sequence
    FROM entries
    WHERE account_id = sqlc.arg('account_id')
)
INSERT INTO balance_snapshots (account_id, balance, last_entry_sequence)
SELECT sqlc.arg('account_id'),
    balance,
    last_sequence
FROM current_balance
RETURNING *;
-- name: GetLatestBalanceSnapshot :one
SELECT *
FROM balance_snapshots
WHERE account_id = sqlc.arg('account_id')
ORDER BY last_entry_sequence DESC
LIMIT 1;