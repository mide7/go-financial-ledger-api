DROP TRIGGER IF EXISTS trg_enforce_transactions_immutability ON transactions;
DROP TRIGGER IF EXISTS trg_enforce_entries_immutability ON entries;
DROP FUNCTION IF EXISTS func_prevent_ledger_modification();