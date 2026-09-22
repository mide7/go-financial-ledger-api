-- name: InsertDiscrepancyRecord :one
INSERT INTO discrepancy_records (
        account_id,
        raw_ledger_sum,
        snapshot_balance,
        calculated_total,
        discrepancy_amount,
        last_snapshot_sequence,
        current_max_sequence
    )
VALUES (
        sqlc.arg('account_id'),
        sqlc.arg('raw_ledger_sum'),
        sqlc.arg('snapshot_balance'),
        sqlc.arg('calculated_total'),
        sqlc.arg('discrepancy_amount'),
        sqlc.arg('last_snapshot_sequence'),
        sqlc.arg('current_max_sequence')
    )
RETURNING *;
-- name: ListUnresolvedDiscrepancies :many
SELECT *
FROM discrepancy_records
WHERE resolved_at IS NULL
ORDER BY detected_at DESC;
-- name: ResolveDiscrepancy :exec
UPDATE discrepancy_records
SET resolved_at = NOW(),
    resolution_notes = sqlc.arg('notes')
WHERE id = sqlc.arg('id');