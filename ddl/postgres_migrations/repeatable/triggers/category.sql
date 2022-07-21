DROP TRIGGER IF EXISTS category_last_update_before_update_trg ON category;
CREATE TRIGGER category_last_update_before_update_trg BEFORE UPDATE ON category
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
