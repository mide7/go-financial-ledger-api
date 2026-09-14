-- Core Ledger Lookup & Composite Indexes
CREATE INDEX IF NOT EXISTS idx_entries_account_id ON entries(account_id);
CREATE INDEX IF NOT EXISTS idx_entries_transaction_id ON entries(transaction_id);
-- Covering Index for Fast Point-in-Time Balance Calculations
CREATE INDEX IF NOT EXISTS idx_entries_account_sequence_type_amount ON entries(account_id, entry_sequence) INCLUDE (type, amount);
-- Transaction Indexes
CREATE INDEX IF NOT EXISTS idx_transactions_currency_created_at ON transactions(currency, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_parent_id ON transactions(parent_transaction_id)
WHERE parent_transaction_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_transactions_metadata ON transactions USING gin (metadata);
-- Operational & Cleanup Indexes
CREATE INDEX IF NOT EXISTS idx_idempotency_expires_at ON idempotency_records(expires_at);
CREATE INDEX IF NOT EXISTS idx_idempotency_active ON idempotency_records(expires_at)
WHERE status = 'IN_PROGRESS';
CREATE INDEX IF NOT EXISTS idx_balance_snapshots_account_seq_desc ON balance_snapshots(account_id, last_entry_sequence DESC);