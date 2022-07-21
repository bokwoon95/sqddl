CREATE OR ALTER TRIGGER rental_last_update_after_update_trg ON rental AFTER UPDATE AS
UPDATE rental
SET last_update = CURRENT_TIMESTAMP
FROM rental
JOIN INSERTED ON INSERTED.rental_id = rental.rental_id;
