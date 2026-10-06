DROP TRIGGER IF EXISTS trg_enforce_account_status_history_immutability ON account_status_history;
DROP TRIGGER IF EXISTS trg_enforce_transactions_immutability ON transactions;
DROP TRIGGER IF EXISTS trg_enforce_entries_immutability ON entries;
DROP FUNCTION IF EXISTS func_prevent_record_modification();