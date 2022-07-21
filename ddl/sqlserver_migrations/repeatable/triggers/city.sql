CREATE OR ALTER TRIGGER city_last_update_after_update_trg ON city AFTER UPDATE AS
UPDATE city
SET last_update = CURRENT_TIMESTAMP
FROM city
JOIN INSERTED ON INSERTED.city_id = city.city_id;
