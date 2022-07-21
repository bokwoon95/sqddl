CREATE OR ALTER TRIGGER store_last_update_after_update_trg ON store AFTER UPDATE AS
UPDATE store
SET last_update = CURRENT_TIMESTAMP
FROM store
JOIN INSERTED ON INSERTED.store_id = store.store_id;
