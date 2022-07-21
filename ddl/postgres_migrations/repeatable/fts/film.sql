DROP TRIGGER IF EXISTS film_fulltext_before_insert_update_trg ON film;
CREATE TRIGGER film_fulltext_before_insert_update_trg BEFORE INSERT OR UPDATE ON film
FOR EACH ROW EXECUTE PROCEDURE tsvector_update_trigger(fulltext, 'pg_catalog.english', title, description);
