DROP TRIGGER IF EXISTS film_category_last_update_before_update_trg ON film_category;
CREATE TRIGGER film_category_last_update_before_update_trg BEFORE UPDATE ON film_category
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
