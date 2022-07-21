DROP TRIGGER IF EXISTS store_last_update_after_update_trg;
CREATE TRIGGER store_last_update_after_update_trg AFTER UPDATE ON store BEGIN
    UPDATE store SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
