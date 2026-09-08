CREATE INDEX IF NOT EXISTS idx_entries_account_id ON entries(account_id);
CREATE INDEX IF NOT EXISTS idx_entries_transaction_id ON entries(transaction_id);
CREATE INDEX IF NOT EXISTS idx_entries_account_sequence ON entries(account_id, entry_sequence);
CREATE INDEX IF NOT EXISTS idx_idempotency_expires_at ON idempotency_records(expires_at);
CREATE INDEX IF NOT EXISTS idx_balance_snapshots_account_seq_desc ON balance_snapshots(account_id, last_entry_sequence DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_metadata ON transactions USING gin (metadata);