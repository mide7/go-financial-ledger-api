CREATE OR REPLACE FUNCTION func_check_transaction_integrity() RETURNS TRIGGER AS $$
DECLARE invalid_tx UUID;
BEGIN -- Check 1: Ensure all entry account currencies match the parent transaction currency
SELECT ne.transaction_id INTO invalid_tx
FROM new_entries ne
    JOIN accounts a ON ne.account_id = a.id
    JOIN transactions t ON ne.transaction_id = t.id
WHERE a.currency != t.currency
LIMIT 1;
-- Custom SQLSTATEs (class LD, "ledger") let the application map violations
-- without matching message text: LD001 currency mismatch, LD002 unbalanced.
IF invalid_tx IS NOT NULL THEN RAISE EXCEPTION 'Currency Mismatch: Transaction % has entries with account currencies that do not match the transaction header currency.',
invalid_tx USING ERRCODE = 'LD001';
END IF;
-- Check 2: Ensure net balance per transaction_id strictly equals zero
SELECT e.transaction_id INTO invalid_tx
FROM entries e
WHERE e.transaction_id IN (
        SELECT transaction_id
        FROM new_entries
    )
GROUP BY e.transaction_id
HAVING SUM(
        CASE
            WHEN e.type = 'CREDIT' THEN e.amount
            WHEN e.type = 'DEBIT' THEN - e.amount
            ELSE 0
        END
    ) != 0
LIMIT 1;
IF invalid_tx IS NOT NULL THEN RAISE EXCEPTION 'Double-Entry Violation: Transaction % is unbalanced.',
invalid_tx USING ERRCODE = 'LD002';
END IF;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- Statement-level deferred constraint trigger using transition tables
CREATE TRIGGER trg_check_transaction_integrity
AFTER
INSERT ON entries REFERENCING NEW TABLE AS new_entries FOR EACH STATEMENT EXECUTE FUNCTION func_check_transaction_integrity();
-- CREATE CONSTRAINT TRIGGER trg_check_transaction_integrity
-- AFTER
-- INSERT ON entries REFERENCING NEW TABLE AS new_entries DEFERRABLE INITIALLY DEFERRED FOR EACH STATEMENT EXECUTE FUNCTION func_check_transaction_integrity();