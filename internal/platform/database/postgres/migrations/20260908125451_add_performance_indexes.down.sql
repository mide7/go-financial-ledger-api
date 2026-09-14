-- Operational & Cleanup Indexes
DROP INDEX IF EXISTS idx_balance_snapshots_account_seq_desc;
DROP INDEX IF EXISTS idx_idempotency_active;
DROP INDEX IF EXISTS idx_idempotency_expires_at;
-- Transaction Indexes
DROP INDEX IF EXISTS idx_transactions_metadata;
DROP INDEX IF EXISTS idx_transactions_parent_id;
DROP INDEX IF EXISTS idx_transactions_currency_created_at;
-- Covering Index for Fast Point-in-Time Balance Calculations
DROP INDEX IF EXISTS idx_entries_account_sequence_type_amount;
-- Core Ledger Lookup & Composite Indexes
DROP INDEX IF EXISTS idx_entries_transaction_id;
DROP INDEX IF EXISTS idx_entries_account_id;