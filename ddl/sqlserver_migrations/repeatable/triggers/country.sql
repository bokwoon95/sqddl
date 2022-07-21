CREATE OR ALTER TRIGGER country_last_update_after_update_trg ON country AFTER UPDATE AS
UPDATE country
SET last_update = CURRENT_TIMESTAMP
FROM country
JOIN INSERTED ON INSERTED.country_id = country.country_id;
