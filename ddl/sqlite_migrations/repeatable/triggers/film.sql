DROP TRIGGER IF EXISTS film_last_update_after_update_trg;
CREATE TRIGGER film_last_update_after_update_trg AFTER UPDATE ON film BEGIN
    UPDATE film SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
