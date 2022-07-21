DROP TRIGGER IF EXISTS inventory_last_update_before_update_trg ON inventory;
CREATE TRIGGER inventory_last_update_before_update_trg BEFORE UPDATE ON inventory
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
