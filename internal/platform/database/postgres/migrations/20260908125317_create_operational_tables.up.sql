-- 4. Idempotency Table
CREATE TABLE IF NOT EXISTS idempotency_records (
    key VARCHAR(256) PRIMARY KEY,
    request_hash VARCHAR(64) NOT NULL,
    -- 'IN_PROGRESS', 'COMPLETED', 'FAILED'
    status VARCHAR(32) NOT NULL DEFAULT 'IN_PROGRESS',
    status_code INT NULL,
    response_body JSONB NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    --
    CONSTRAINT chk_idempotency_status CHECK (
    status IN ('IN_PROGRESS', 'COMPLETED', 'FAILED')
)
);
-- 5. Balance Snapshots Table
CREATE TABLE IF NOT EXISTS balance_snapshots (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    balance BIGINT NOT NULL,
    last_entry_sequence BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    --
    CONSTRAINT uq_account_sequence UNIQUE (account_id, last_entry_sequence)
);