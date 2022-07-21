DROP TRIGGER IF EXISTS film_last_update_before_update_trg ON film;
CREATE TRIGGER film_last_update_before_update_trg BEFORE UPDATE ON film
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
