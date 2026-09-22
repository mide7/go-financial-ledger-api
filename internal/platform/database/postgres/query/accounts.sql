-- name: CreateAccount :one
INSERT INTO accounts (owner_id, type, currency)
VALUES ($1, $2, $3)
RETURNING *;
-- name: GetAccount :one
SELECT *
FROM accounts
WHERE id = $1
LIMIT 1;
-- name: ListAccounts :many
SELECT *
FROM accounts
WHERE (
        sqlc.narg(owner_id)::text IS NULL
        OR owner_id = sqlc.narg(owner_id)
    )
    AND (
        sqlc.narg(type)::text IS NULL
        OR type = sqlc.narg(type)
    )
    AND (
        sqlc.narg(currency)::text IS NULL
        OR currency = sqlc.narg(currency)
    )
ORDER BY -- Sort by created_at
    CASE
        WHEN sqlc.narg(sort_by)::text = 'created_at'
        AND LOWER(sqlc.narg(sort_order)::text) = 'asc' THEN created_at
    END ASC,
    CASE
        WHEN sqlc.narg(sort_by)::text = 'created_at'
        AND LOWER(sqlc.narg(sort_order)::text) = 'desc' THEN created_at
    END DESC,
    -- Sort by type
    CASE
        WHEN sqlc.narg(sort_by)::text = 'type'
        AND LOWER(sqlc.narg(sort_order)::text) = 'asc' THEN type
    END ASC,
    CASE
        WHEN sqlc.narg(sort_by)::text = 'type'
        AND LOWER(sqlc.narg(sort_order)::text) = 'desc' THEN type
    END DESC,
    -- Sort by currency
    CASE
        WHEN sqlc.narg(sort_by)::text = 'currency'
        AND LOWER(sqlc.narg(sort_order)::text) = 'asc' THEN currency
    END ASC,
    CASE
        WHEN sqlc.narg(sort_by)::text = 'currency'
        AND LOWER(sqlc.narg(sort_order)::text) = 'desc' THEN currency
    END DESC,
    -- Default fallback if sort_by is empty or invalid
    CASE
        WHEN sqlc.narg(sort_by)::text IS NULL
        OR sqlc.narg(sort_by)::text = '' THEN created_at
    END DESC
LIMIT sqlc.arg('limit')::int OFFSET sqlc.arg('offset')::int;
-- name: GetAccountBalance :one
WITH account_info AS (
    SELECT a.id AS account_id,
        a.currency,
        c.exponent
    FROM accounts a
        JOIN currencies c ON a.currency = c.code
    WHERE a.id = sqlc.arg('account_id')
),
latest_snapshot AS (
    SELECT balance,
        last_entry_sequence
    FROM balance_snapshots
    WHERE account_id = sqlc.arg('account_id')
    ORDER BY last_entry_sequence DESC
    LIMIT 1
), baseline AS (
    SELECT balance,
        last_entry_sequence
    FROM latest_snapshot
    UNION ALL
    -- Fallback zero-state if no snapshot exists yet for a new account
    SELECT 0 AS balance,
        0 AS last_entry_sequence
    LIMIT 1
), raw_balance AS (
    SELECT (
            COALESCE(b.balance, 0) + COALESCE(
                SUM(
                    CASE
                        WHEN e.type = 'CREDIT' THEN e.amount
                        WHEN e.type = 'DEBIT' THEN - e.amount
                        ELSE 0
                    END
                ),
                0
            )
        )::BIGINT AS minor_units
    FROM baseline b
        LEFT JOIN entries e ON e.account_id = sqlc.arg('account_id')
        AND e.entry_sequence > b.last_entry_sequence
    GROUP BY b.balance
)
SELECT ai.currency,
    ai.exponent,
    rb.minor_units AS balance_minor,
    (rb.minor_units::NUMERIC / POWER(10, ai.exponent))::NUMERIC AS balance_major
FROM account_info ai
    CROSS JOIN raw_balance rb;
-- name: GetAccountStatementSummary :one
WITH account_info AS (
    SELECT a.id AS account_id,
        a.currency,
        c.exponent
    FROM accounts a
        JOIN currencies c ON a.currency = c.code
    WHERE a.id = sqlc.arg('account_id')
),
prior_activity AS (
    SELECT COALESCE(
            SUM(
                CASE
                    WHEN type = 'DEBIT' THEN amount
                    ELSE - amount
                END
            ),
            0
        )::BIGINT AS balance
    FROM entries
    WHERE account_id = sqlc.arg('account_id')
        AND (
            sqlc.narg(start_date)::timestamptz IS NOT NULL
            AND created_at < sqlc.narg(start_date)
        )
),
period_activity AS (
    SELECT COALESCE(
            SUM(
                CASE
                    WHEN type = 'DEBIT' THEN amount
                    ELSE 0
                END
            ),
            0
        )::BIGINT AS total_debits,
        COALESCE(
            SUM(
                CASE
                    WHEN type = 'CREDIT' THEN amount
                    ELSE 0
                END
            ),
            0
        )::BIGINT AS total_credits
    FROM entries
    WHERE account_id = sqlc.arg('account_id')
        AND (
            sqlc.narg(start_date)::timestamptz IS NULL
            OR created_at >= sqlc.narg(start_date)
        )
        AND (
            sqlc.narg(end_date)::timestamptz IS NULL
            OR created_at <= sqlc.narg(end_date)
        )
)
SELECT ai.account_id,
    ai.currency,
    ai.exponent,
    p.balance AS opening_balance,
    pa.total_debits,
    pa.total_credits,
    (p.balance + pa.total_debits - pa.total_credits)::BIGINT AS closing_balance
FROM account_info ai
    CROSS JOIN prior_activity p
    CROSS JOIN period_activity pa;
WITH base_snapshot AS (
    SELECT bs.balance,
        bs.last_entry_sequence
    FROM balance_snapshots bs
        JOIN entries se ON se.entry_sequence = bs.last_entry_sequence
    WHERE bs.account_id = sqlc.arg('account_id')
        AND se.created_at <= sqlc.arg('start_date')
    ORDER BY bs.last_entry_sequence DESC
    LIMIT 1
), account_history AS (
    SELECT e.id AS entry_id,
        e.transaction_id,
        e.type AS entry_type,
        e.amount,
        t.description,
        e.created_at AS entry_created_at,
        -- explicit alias, always
        e.entry_sequence,
        COALESCE(
            (
                SELECT balance
                FROM base_snapshot
            ),
            0
        ) + SUM(
            CASE
                WHEN e.type = 'DEBIT' THEN e.amount
                ELSE - e.amount
            END
        ) OVER (
            ORDER BY e.entry_sequence ASC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
        ) AS running_balance
    FROM entries e
        JOIN transactions t ON e.transaction_id = t.id
    WHERE e.account_id = sqlc.arg('account_id')
        AND e.entry_sequence > COALESCE(
            (
                SELECT last_entry_sequence
                FROM base_snapshot
            ),
            0
        )
        AND e.created_at <= sqlc.arg('end_date')
)
SELECT entry_id,
    transaction_id,
    entry_type,
    amount,
    running_balance,
    description,
    entry_sequence,
    entry_created_at AS created_at
FROM account_history
WHERE entry_created_at >= sqlc.arg('start_date')
ORDER BY entry_sequence DESC
LIMIT sqlc.arg('limit')::int OFFSET sqlc.arg('offset')::int;
-- TODO: GetAccountStatementEntries
-- name: ReconcileAccount :one
WITH account_currency AS (
    SELECT a.id AS account_id,
        a.currency,
        c.exponent
    FROM accounts a
        JOIN currencies c ON c.code = a.currency
    WHERE a.id = sqlc.arg('account_id')
),
latest_snapshot AS (
    SELECT bs.balance,
        bs.last_entry_sequence
    FROM balance_snapshots bs
    WHERE bs.account_id = sqlc.arg('account_id')
    ORDER BY bs.last_entry_sequence DESC
    LIMIT 1
), raw_ledger AS (
    SELECT COALESCE(
            SUM(
                CASE
                    WHEN e.type = 'DEBIT' THEN e.amount
                    ELSE - e.amount
                END
            ),
            0
        )::bigint AS raw_sum,
        COALESCE(MAX(e.entry_sequence), 0)::bigint AS max_sequence
    FROM entries e
    WHERE e.account_id = sqlc.arg('account_id')
),
delta AS (
    SELECT COALESCE(
            SUM(
                CASE
                    WHEN e.type = 'DEBIT' THEN e.amount
                    ELSE - e.amount
                END
            ),
            0
        )::bigint AS delta_sum
    FROM entries e
    WHERE e.account_id = sqlc.arg('account_id')
        AND e.entry_sequence > COALESCE(
            (
                SELECT last_entry_sequence
                FROM latest_snapshot
            ),
            0
        )
)
SELECT ac.account_id,
    CASE
        WHEN rl.raw_sum = COALESCE(ls.balance, 0) + d.delta_sum THEN 'RECONCILED'
        ELSE 'DISCREPANCY_DETECTED'
    END AS status,
    ac.currency,
    ac.exponent,
    rl.raw_sum AS raw_ledger_sum,
    COALESCE(ls.balance, 0)::bigint AS snapshot_balance,
    d.delta_sum AS delta_since_snapshot,
    (COALESCE(ls.balance, 0) + d.delta_sum)::bigint AS calculated_total,
    (
        rl.raw_sum - (COALESCE(ls.balance, 0) + d.delta_sum)
    )::bigint AS discrepancy_amount,
    COALESCE(ls.last_entry_sequence, 0)::bigint AS last_snapshot_sequence,
    rl.max_sequence AS current_max_sequence,
    NOW()::timestamptz AS reconciled_at
FROM account_currency ac
    CROSS JOIN raw_ledger rl
    CROSS JOIN delta d
    LEFT JOIN latest_snapshot ls ON true;
-- TODO: Rewrite these queries to use the new account_status_history table
-- -- name: FreezeAccount :exec
-- UPDATE accounts
-- SET is_frozen = true,
--     frozen_reason = sqlc.arg('reason'),
--     frozen_at = NOW(),
--     updated_at = NOW()
-- WHERE id = sqlc.arg('account_id');
-- -- name: UnfreezeAccount :exec
-- UPDATE accounts
-- SET is_frozen = false,
--     frozen_reason = NULL,
--     frozen_at = NULL,
--     updated_at = NOW()
-- WHERE id = sqlc.arg('account_id');
-- -- name: GetFrozenAccountsInSet :many
-- SELECT id
-- FROM accounts
-- WHERE id = ANY(sqlc.arg('account_ids')::uuid [])
--     AND is_frozen = true;