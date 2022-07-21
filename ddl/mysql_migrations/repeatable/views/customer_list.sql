CREATE OR REPLACE VIEW customer_list AS
SELECT
    cu.customer_id AS id
    ,CONCAT(cu.first_name, ' ', cu.last_name) AS name
    ,a.address, a.postal_code AS `zip code`
    ,a.phone
    ,city.city
    ,country.country
    ,CASE WHEN cu.active THEN 'active' ELSE '' END AS notes
    ,cu.store_id AS sid
FROM
    customer AS cu
    JOIN address AS a ON a.address_id = cu.address_id
    JOIN city ON city.city_id = a.city_id
    JOIN country ON country.country_id = city.country_id
;
