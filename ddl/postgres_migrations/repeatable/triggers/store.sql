DROP TRIGGER IF EXISTS store_last_update_before_update_trg ON store;
CREATE TRIGGER store_last_update_before_update_trg BEFORE UPDATE ON store
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
