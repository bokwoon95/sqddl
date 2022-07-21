CREATE OR ALTER TRIGGER address_last_update_after_update_trg ON address AFTER UPDATE AS
UPDATE address
SET last_update = CURRENT_TIMESTAMP
FROM address
JOIN INSERTED ON INSERTED.address_id = address.address_id;
