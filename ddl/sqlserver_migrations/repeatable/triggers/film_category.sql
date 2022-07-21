CREATE OR ALTER TRIGGER film_category_last_update_after_update_trg ON film_category AFTER UPDATE AS
UPDATE film_category
SET last_update = CURRENT_TIMESTAMP
FROM film_category
JOIN INSERTED ON INSERTED.film_id = film_category.film_id AND INSERTED.category_id = film_category.category_id;
