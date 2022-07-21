DROP TRIGGER IF EXISTS language_last_update_after_update_trg;
CREATE TRIGGER language_last_update_after_update_trg AFTER UPDATE ON language BEGIN
    UPDATE language SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
