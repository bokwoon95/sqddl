CREATE MATERIALIZED VIEW IF NOT EXISTS full_address AS
SELECT
    country.country_id
    ,city.city_id
    ,address.address_id
    ,country.country
    ,city.city
    ,address.address
    ,address.address2
    ,address.district
    ,address.postal_code
    ,address.phone
    ,address.last_update
FROM
    address
    JOIN city ON city.city_id = address.city_id
    JOIN country ON country.country_id = city.country_id
;

CREATE UNIQUE INDEX IF NOT EXISTS full_address_address_id_idx ON full_address (address_id);

CREATE OR REPLACE FUNCTION refresh_full_address() RETURNS trigger AS $$ BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY full_address;
    RETURN NULL;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS address_refresh_full_address_trg ON address;
CREATE TRIGGER address_refresh_full_address_trg AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON address
FOR EACH STATEMENT EXECUTE PROCEDURE refresh_full_address();

DROP TRIGGER IF EXISTS city_refresh_full_address_trg ON city;
CREATE TRIGGER city_refresh_full_address_trg AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON city
FOR EACH STATEMENT EXECUTE PROCEDURE refresh_full_address();

DROP TRIGGER IF EXISTS country_refresh_full_address_trg ON country;
CREATE TRIGGER country_refresh_full_address_trg AFTER INSERT OR UPDATE OR DELETE OR TRUNCATE ON country
FOR EACH STATEMENT EXECUTE PROCEDURE refresh_full_address();
