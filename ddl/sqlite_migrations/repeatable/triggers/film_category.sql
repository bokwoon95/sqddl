DROP TRIGGER IF EXISTS film_category_last_update_after_update_trg;
CREATE TRIGGER film_category_last_update_after_update_trg AFTER UPDATE ON film_category BEGIN
    UPDATE film_category SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
