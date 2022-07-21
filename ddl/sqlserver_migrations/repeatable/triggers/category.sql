CREATE OR ALTER TRIGGER category_last_update_after_update_trg ON category AFTER UPDATE AS
UPDATE category
SET last_update = CURRENT_TIMESTAMP
FROM category
JOIN INSERTED ON INSERTED.category_id = category.category_id;
