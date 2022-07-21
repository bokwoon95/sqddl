DROP TRIGGER IF EXISTS address_last_update_before_update_trg ON address;
CREATE TRIGGER address_last_update_before_update_trg BEFORE UPDATE ON address
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
