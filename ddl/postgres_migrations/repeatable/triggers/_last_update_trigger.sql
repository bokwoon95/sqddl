CREATE OR REPLACE FUNCTION last_update_trg() RETURNS trigger AS $$ BEGIN
    NEW.last_update = NOW();
    RETURN NEW;
END; $$ LANGUAGE plpgsql;
