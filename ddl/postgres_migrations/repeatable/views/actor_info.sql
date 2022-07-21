CREATE OR REPLACE VIEW actor_info AS
SELECT
    a.actor_id
    ,a.first_name
    ,a.last_name
    ,jsonb_object_agg(c.name, (
        SELECT
            jsonb_agg(f.title)
        FROM
            film AS f
            JOIN film_category AS fc ON fc.film_id = f.film_id
            JOIN film_actor AS fa ON fa.film_id = f.film_id
        WHERE
            fc.category_id = c.category_id
            AND fa.actor_id = a.actor_id
        GROUP BY
            fa.actor_id
    )) AS film_info
FROM
    actor AS a
    LEFT JOIN film_actor AS fa ON fa.actor_id = a.actor_id
    LEFT JOIN film_category AS fc ON fc.film_id = fa.film_id
    LEFT JOIN category AS c ON c.category_id = fc.category_id
GROUP BY
    a.actor_id
    ,a.first_name
    ,a.last_name
;
