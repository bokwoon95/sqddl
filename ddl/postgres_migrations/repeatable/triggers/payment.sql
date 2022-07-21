DROP TRIGGER IF EXISTS payment_last_update_before_update_trg ON payment;
CREATE TRIGGER payment_last_update_before_update_trg BEFORE UPDATE ON payment
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
