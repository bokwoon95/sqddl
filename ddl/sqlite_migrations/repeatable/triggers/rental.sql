DROP TRIGGER IF EXISTS rental_last_update_after_update_trg;
CREATE TRIGGER rental_last_update_after_update_trg AFTER UPDATE ON rental BEGIN
    UPDATE rental SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
