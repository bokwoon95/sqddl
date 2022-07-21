DROP TRIGGER IF EXISTS staff_last_update_after_update_trg;
CREATE TRIGGER staff_last_update_after_update_trg AFTER UPDATE ON staff BEGIN
    UPDATE staff SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
