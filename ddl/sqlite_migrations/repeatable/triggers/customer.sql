DROP TRIGGER IF EXISTS customer_last_update_after_update_trg;
CREATE TRIGGER customer_last_update_after_update_trg AFTER UPDATE ON customer BEGIN
    UPDATE customer SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
