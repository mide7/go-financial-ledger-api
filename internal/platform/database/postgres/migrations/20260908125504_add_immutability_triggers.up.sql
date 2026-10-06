CREATE OR REPLACE FUNCTION func_prevent_record_modification() RETURNS TRIGGER AS $$ BEGIN RAISE EXCEPTION 'Records are strictly immutable. Updates and deletes are forbidden.';
END;
$$ LANGUAGE plpgsql;
--
CREATE TRIGGER trg_enforce_entries_immutability BEFORE
UPDATE
    OR DELETE ON entries FOR EACH ROW EXECUTE FUNCTION func_prevent_record_modification();
--
CREATE TRIGGER trg_enforce_transactions_immutability BEFORE
UPDATE
    OR DELETE ON transactions FOR EACH ROW EXECUTE FUNCTION func_prevent_record_modification();
--
CREATE TRIGGER trg_enforce_account_status_history_immutability BEFORE
UPDATE
    OR DELETE ON account_status_history FOR EACH ROW EXECUTE FUNCTION func_prevent_record_modification();