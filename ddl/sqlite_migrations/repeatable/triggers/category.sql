DROP TRIGGER IF EXISTS category_last_update_after_update_trg;
CREATE TRIGGER category_last_update_after_update_trg AFTER UPDATE ON category BEGIN
    UPDATE category SET last_update = unixepoch() WHERE ROWID = NEW.ROWID;
END;
