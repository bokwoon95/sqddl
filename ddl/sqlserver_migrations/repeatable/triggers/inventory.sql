CREATE OR ALTER TRIGGER inventory_last_update_after_update_trg ON inventory AFTER UPDATE AS
UPDATE inventory
SET last_update = CURRENT_TIMESTAMP
FROM inventory
JOIN INSERTED ON INSERTED.inventory_id = inventory.inventory_id;
