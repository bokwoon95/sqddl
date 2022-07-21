DROP TRIGGER IF EXISTS customer_last_update_before_update_trg ON customer;
CREATE TRIGGER customer_last_update_before_update_trg BEFORE UPDATE ON customer
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
