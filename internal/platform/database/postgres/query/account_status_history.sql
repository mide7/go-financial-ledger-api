-- name: InsertStatusHistory :one
INSERT INTO account_status_history (
        account_id,
        previous_status,
        new_status,
        reason,
        actor_type,
        actor_id
    )
VALUES (
        sqlc.arg('account_id'),
        sqlc.arg('previous_status'),
        sqlc.arg('new_status'),
        sqlc.arg('reason'),
        sqlc.arg('actor_type'),
        sqlc.arg('actor_id')
    )
RETURNING *;
-- name: ListAccountStatusHistory :many
SELECT *
FROM account_status_history
WHERE account_id = sqlc.arg('account_id')
    AND (
        sqlc.narg('cursor_created_at')::timestamptz IS NULL
        OR (created_at, id) < (
            sqlc.narg('cursor_created_at')::timestamptz,
            sqlc.narg('cursor_id')::uuid
        )
    )
ORDER BY created_at DESC,
    id DESC
LIMIT sqlc.arg('limit')::int;