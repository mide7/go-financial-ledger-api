-- 4. Account Status History Table
CREATE TABLE IF NOT EXISTS account_status_history (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    previous_status VARCHAR(32) NOT NULL,
    new_status VARCHAR(32) NOT NULL,
    reason TEXT NOT NULL,
    actor_type VARCHAR(32) NOT NULL,
    -- 'SYSTEM', 'ADMIN_USER', 'RISK_ENGINE'
    actor_id VARCHAR(64) NOT NULL,
    -- ID of the admin or system service triggering the freeze
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_status_history_actors CHECK (
        actor_type IN ('SYSTEM', 'ADMIN_USER', 'RISK_ENGINE')
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
-- 6. Discrepancy Records Table
CREATE TABLE IF NOT EXISTS discrepancy_records (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    raw_ledger_sum BIGINT NOT NULL,
    snapshot_balance BIGINT NOT NULL,
    calculated_total BIGINT NOT NULL,
    discrepancy_amount BIGINT NOT NULL,
    last_snapshot_sequence BIGINT NOT NULL,
    current_max_sequence BIGINT NOT NULL,
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolution_notes TEXT
);
-- 7. Idempotency Table
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