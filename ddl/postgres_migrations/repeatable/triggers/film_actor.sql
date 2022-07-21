DROP TRIGGER IF EXISTS film_actor_last_update_before_update_trg ON film_actor;
CREATE TRIGGER film_actor_last_update_before_update_trg BEFORE UPDATE ON film_actor
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
