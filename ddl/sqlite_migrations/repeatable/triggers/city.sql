DROP TRIGGER IF EXISTS city_last_update_after_update_trg;
CREATE TRIGGER city_last_update_after_update_trg AFTER UPDATE ON city BEGIN
    UPDATE city SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
