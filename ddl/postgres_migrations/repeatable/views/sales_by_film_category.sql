CREATE OR REPLACE VIEW sales_by_film_category AS
SELECT
    c.name AS category
    ,SUM(p.amount) AS total_sales
FROM
    payment AS p
    JOIN rental AS r ON r.rental_id = p.rental_id
    JOIN inventory AS i ON i.inventory_id = r.inventory_id
    JOIN film AS f ON f.film_id = i.film_id
    JOIN film_category AS fc ON fc.film_id = f.film_id
    JOIN category AS c ON c.category_id = fc.category_id
GROUP BY
    c.name
;
