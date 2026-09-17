-- 1. Create notification trigger function
CREATE OR REPLACE FUNCTION notify_currency_change() RETURNS trigger AS $$ BEGIN -- Broadcast notification channel 'currency_updated' with the affected currency code
    PERFORM pg_notify('currency_updated', COALESCE(NEW.code, OLD.code));
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- 2. Attach trigger to currencies table for writes
CREATE TRIGGER trg_currency_updated
AFTER
INSERT
    OR
UPDATE
    OR DELETE ON currencies FOR EACH ROW EXECUTE FUNCTION notify_currency_change();