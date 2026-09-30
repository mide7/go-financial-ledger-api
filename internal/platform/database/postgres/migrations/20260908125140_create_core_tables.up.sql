CREATE TYPE entry_type AS ENUM ('DEBIT', 'CREDIT');
-- 1. Currency Table
CREATE TABLE IF NOT EXISTS currencies (
    -- ISO 4217 (e.g., 'NGN', 'USD')
    code VARCHAR(3) PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    -- Minor unit digits
    exponent SMALLINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- 2. Account Types Table
CREATE TABLE account_types (
    type VARCHAR(64) PRIMARY KEY,
    nature VARCHAR(16) NOT NULL CHECK (
        nature IN (
            'ASSET',
            'LIABILITY',
            'REVENUE',
            'EXPENSE',
            'EQUITY'
        )
    ),
    normal_balance entry_type NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);
-- 3. Accounts Table
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    owner_id VARCHAR(64) NOT NULL,
    type VARCHAR(64) NOT NULL REFERENCES account_types(type) ON DELETE RESTRICT,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN' REFERENCES currencies(code) ON DELETE RESTRICT,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_account_status CHECK (
        status IN (
            -- Can receive funds and transfer out
            'ACTIVE',
            -- Cannot receive funds or transfer out
            'FULLY_FROZEN',
            -- Can receive funds, cannot transfer out
            'DEBIT_RESTRICTED',
            -- Can transfer out, cannot receive funds
            'CREDIT_RESTRICTED',
            -- Can neither receive nor transfer out, permanently closed
            'CLOSED'
        )
    ),
    CONSTRAINT uq_owner_type_currency UNIQUE (owner_id, type, currency)
);
-- 4. Transactions Table 
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    reference VARCHAR(128) UNIQUE NOT NULL,
    -- e.g., 'TICKET_PURCHASE', 'ESCROW_RELEASE', 'REFUND'
    type VARCHAR(64) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'NGN' REFERENCES currencies(code) ON DELETE RESTRICT,
    parent_transaction_id UUID NULL REFERENCES transactions(id) ON DELETE RESTRICT,
    description TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    transaction_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- 5. Ledger Entries Table
CREATE TABLE IF NOT EXISTS entries (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE RESTRICT,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    type entry_type NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    entry_sequence BIGSERIAL NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);