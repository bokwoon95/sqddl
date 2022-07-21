DROP TRIGGER IF EXISTS language_last_update_before_update_trg ON language;
CREATE TRIGGER language_last_update_before_update_trg BEFORE UPDATE ON language
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
