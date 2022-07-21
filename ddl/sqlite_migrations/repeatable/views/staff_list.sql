DROP VIEW IF EXISTS staff_list;
CREATE VIEW staff_list AS
SELECT
    s.staff_id AS id
    ,s.first_name || ' ' || s.last_name AS name
    ,a.address, a.postal_code AS "zip code"
    ,a.phone
    ,ci.city
    ,co.country
    ,s.store_id AS sid
FROM
    staff AS s
    JOIN address AS a ON a.address_id = s.address_id
    JOIN city AS ci ON ci.city_id = a.city_id
    JOIN country AS co ON co.country_id = ci.country_id
;
