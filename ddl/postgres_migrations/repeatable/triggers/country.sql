DROP TRIGGER IF EXISTS country_last_update_before_update_trg ON country;
CREATE TRIGGER country_last_update_before_update_trg BEFORE UPDATE ON country
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
