CREATE OR ALTER TRIGGER language_last_update_after_update_trg ON language AFTER UPDATE AS
UPDATE language
SET last_update = CURRENT_TIMESTAMP
FROM language
JOIN INSERTED ON INSERTED.language_id = language.language_id;
