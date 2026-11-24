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
-- Newest first. ids are UUIDv7 and detected_at defaults to the insert time,
-- so id order is detection order; the cursor is the last id of the previous page.
SELECT *
FROM discrepancy_records
WHERE resolved_at IS NULL
    AND (
        sqlc.narg('cursor')::uuid IS NULL
        OR id < sqlc.narg('cursor')::uuid
    )
ORDER BY id DESC
LIMIT sqlc.arg('limit')::int;
-- name: ResolveDiscrepancy :execrows
UPDATE discrepancy_records
SET resolved_at = NOW(),
    resolution_notes = sqlc.arg('notes')
WHERE id = sqlc.arg('id')
    AND resolved_at IS NULL;