DROP TRIGGER IF EXISTS rental_last_update_before_update_trg ON rental;
CREATE TRIGGER rental_last_update_before_update_trg BEFORE UPDATE ON rental
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
