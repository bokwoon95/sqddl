CREATE OR ALTER TRIGGER customer_last_update_after_update_trg ON customer AFTER UPDATE AS
UPDATE customer
SET last_update = CURRENT_TIMESTAMP
FROM customer
JOIN INSERTED ON INSERTED.customer_id = customer.customer_id;
