-- name: SeedCurrency :exec
INSERT INTO currencies (code, name, exponent, is_active)
VALUES ($1, $2, $3, $4) ON CONFLICT (code) DO NOTHING;
-- name: SeedAccountType :exec
INSERT INTO account_types (type, nature, normal_balance, description)
VALUES ($1, $2, $3, $4) ON CONFLICT (type) DO NOTHING;
-- name: SeedSystemAccount :exec
INSERT INTO accounts (owner_id, type, currency)
VALUES ($1, $2, $3) ON CONFLICT (owner_id, type, currency) DO NOTHING;