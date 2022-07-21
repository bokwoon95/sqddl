DROP TRIGGER IF EXISTS city_last_update_before_update_trg ON city;
CREATE TRIGGER city_last_update_before_update_trg BEFORE UPDATE ON city
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
