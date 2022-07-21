DROP TRIGGER IF EXISTS actor_last_update_before_update_trg ON actor;
CREATE TRIGGER actor_last_update_before_update_trg BEFORE UPDATE ON actor
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
