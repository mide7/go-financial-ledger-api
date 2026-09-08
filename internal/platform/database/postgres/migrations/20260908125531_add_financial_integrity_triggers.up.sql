CREATE OR REPLACE FUNCTION func_check_transaction_integrity() RETURNS TRIGGER AS $$
DECLARE invalid_tx UUID;
BEGIN -- Check 1: Ensure NO SINGLE transaction_id spans multiple currencies
SELECT e.transaction_id INTO invalid_tx
FROM entries e
    JOIN accounts a ON e.account_id = a.id
WHERE e.transaction_id IN (
        SELECT transaction_id
        FROM new_entries
    )
GROUP BY e.transaction_id
HAVING COUNT(DISTINCT a.currency) > 1
LIMIT 1;
IF invalid_tx IS NOT NULL THEN RAISE EXCEPTION 'Currency Mismatch: Transaction % contains accounts with different currencies.',
invalid_tx;
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
invalid_tx;
END IF;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- Statement-level deferred constraint trigger using transition tables
CREATE CONSTRAINT TRIGGER trg_check_transaction_integrity
AFTER
INSERT ON entries DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION func_check_transaction_integrity();