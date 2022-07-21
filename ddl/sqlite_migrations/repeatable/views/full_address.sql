DROP VIEW IF EXISTS full_address;
CREATE VIEW full_address AS
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
