DROP TRIGGER IF EXISTS staff_last_update_before_update_trg ON staff;
CREATE TRIGGER staff_last_update_before_update_trg BEFORE UPDATE ON staff
FOR EACH ROW EXECUTE PROCEDURE last_update_trg();
