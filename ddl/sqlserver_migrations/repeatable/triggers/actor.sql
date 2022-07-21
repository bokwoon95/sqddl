CREATE OR ALTER TRIGGER actor_last_update_after_update_trg ON actor AFTER UPDATE AS
UPDATE actor
SET last_update = CURRENT_TIMESTAMP
FROM actor
JOIN INSERTED ON INSERTED.actor_id = actor.actor_id;
