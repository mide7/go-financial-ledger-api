
-- name: ListEntriesByTransactionID :many
SELECT *
FROM entries
WHERE transaction_id = sqlc.arg('transaction_id')
ORDER BY entry_sequence;
