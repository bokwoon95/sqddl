DROP TRIGGER IF EXISTS address_last_update_after_update_trg;
CREATE TRIGGER address_last_update_after_update_trg AFTER UPDATE ON address BEGIN
    UPDATE address SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
