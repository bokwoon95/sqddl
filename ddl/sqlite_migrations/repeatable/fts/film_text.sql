CREATE VIRTUAL TABLE IF NOT EXISTS film_text USING FTS5 (
    title
    ,description
    ,content='film'
    ,content_rowid='film_id'
);

CREATE TRIGGER IF NOT EXISTS film_fts5_after_insert_trg AFTER INSERT ON film BEGIN
    INSERT INTO film_text (ROWID, title, description) VALUES (NEW.film_id, NEW.title, NEW.description);
END;

CREATE TRIGGER IF NOT EXISTS film_fts5_after_delete_trg AFTER DELETE ON film BEGIN
    INSERT INTO film_text (film_text, ROWID, title, description) VALUES ('delete', OLD.film_id, OLD.title, OLD.description);
END;

CREATE TRIGGER IF NOT EXISTS film_fts5_after_update_trg AFTER UPDATE ON film BEGIN
    INSERT INTO film_text (film_text, ROWID, title, description) VALUES ('delete', OLD.film_id, OLD.title, OLD.description);
    INSERT INTO film_text (ROWID, title, description) VALUES (NEW.film_id, NEW.title, NEW.description);
END;
