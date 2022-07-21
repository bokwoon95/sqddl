DROP TRIGGER IF EXISTS inventory_last_update_after_update_trg;
CREATE TRIGGER inventory_last_update_after_update_trg AFTER UPDATE ON inventory BEGIN
    UPDATE inventory SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
