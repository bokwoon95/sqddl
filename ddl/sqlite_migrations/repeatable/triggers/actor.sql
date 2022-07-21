DROP TRIGGER IF EXISTS actor_last_update_after_update_trg;
CREATE TRIGGER actor_last_update_after_update_trg AFTER UPDATE ON actor BEGIN
    UPDATE actor SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
