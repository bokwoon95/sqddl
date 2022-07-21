DROP VIEW IF EXISTS sales_by_store;
CREATE VIEW sales_by_store AS
SELECT
    ci.city || ',' || co.country AS store
    ,m.first_name || ' ' || m.last_name AS manager
    ,SUM(p.amount) AS total_sales
FROM
    payment AS p
    JOIN rental AS r ON r.rental_id = p.rental_id
    JOIN inventory AS i ON i.inventory_id = r.inventory_id
    JOIN store AS s ON s.store_id = i.store_id
    JOIN address AS a ON a.address_id = s.address_id
    JOIN city AS ci ON ci.city_id = a.city_id
    JOIN country AS co ON co.country_id = ci.country_id
    JOIN staff AS m ON m.staff_id = s.manager_staff_id
GROUP BY
    co.country
    ,ci.city
    ,s.store_id
    ,m.first_name
    ,m.last_name
;
