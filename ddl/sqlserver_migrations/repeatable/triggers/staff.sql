CREATE OR ALTER TRIGGER staff_last_update_after_update_trg ON staff AFTER UPDATE AS
UPDATE staff
SET last_update = CURRENT_TIMESTAMP
FROM staff
JOIN INSERTED ON INSERTED.staff_id = staff.staff_id;
