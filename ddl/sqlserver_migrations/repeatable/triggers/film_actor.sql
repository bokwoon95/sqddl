CREATE OR ALTER TRIGGER film_actor_last_update_after_update_trg ON film_actor AFTER UPDATE AS
UPDATE film_actor
SET last_update = CURRENT_TIMESTAMP
FROM film_actor
JOIN INSERTED ON INSERTED.actor_id = film_actor.actor_id AND INSERTED.film_id = film_actor.film_id;
