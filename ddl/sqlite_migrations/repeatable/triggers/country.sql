DROP TRIGGER IF EXISTS country_last_update_after_update_trg;
CREATE TRIGGER country_last_update_after_update_trg AFTER UPDATE ON country BEGIN
    UPDATE country SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
