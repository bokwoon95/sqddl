CREATE OR ALTER TRIGGER film_last_update_after_update_trg ON film AFTER UPDATE AS
UPDATE film
SET last_update = CURRENT_TIMESTAMP
FROM film
JOIN INSERTED ON INSERTED.film_id = film.film_id;
