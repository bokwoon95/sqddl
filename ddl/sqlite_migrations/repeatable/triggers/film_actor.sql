DROP TRIGGER IF EXISTS film_actor_last_update_after_update_trg;
CREATE TRIGGER film_actor_last_update_after_update_trg AFTER UPDATE ON film_actor BEGIN
    UPDATE film_actor SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
