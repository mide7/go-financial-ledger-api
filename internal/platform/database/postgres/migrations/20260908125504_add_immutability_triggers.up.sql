CREATE OR REPLACE FUNCTION func_prevent_ledger_modification() RETURNS TRIGGER AS $$ BEGIN RAISE EXCEPTION 'Ledger records are strictly immutable. Updates and deletes are forbidden.';
END;
$$ LANGUAGE plpgsql;
--
CREATE TRIGGER trg_enforce_entries_immutability BEFORE
UPDATE
    OR DELETE ON entries FOR EACH ROW EXECUTE FUNCTION func_prevent_ledger_modification();
--
CREATE TRIGGER trg_enforce_transactions_immutability BEFORE
UPDATE
    OR DELETE ON transactions FOR EACH ROW EXECUTE FUNCTION func_prevent_ledger_modification();