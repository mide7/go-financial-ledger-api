-- name: CreateBalanceSnapshot :one
-- The stored balance is positive in the account's normal-balance direction
-- (account_types.normal_balance), so GetAccountBalance and ReconcileAccount can
-- add later entries to it with the same sign rule.
WITH current_balance AS (
    SELECT COALESCE(
            SUM(
                CASE
                    WHEN e.type = at.normal_balance THEN e.amount
                    ELSE - e.amount
                END
            ),
            0
        )::bigint AS balance,
        COALESCE(MAX(e.entry_sequence), 0)::bigint AS last_sequence
    FROM accounts a
        JOIN account_types at ON at.type = a.type
        LEFT JOIN entries e ON e.account_id = a.id
    WHERE a.id = sqlc.arg('account_id')
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
