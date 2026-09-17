-- name: ListActiveCurrencies :many
SELECT code,
    name,
    exponent,
    is_active
FROM currencies
WHERE is_active = true;