CREATE OR ALTER VIEW nicer_but_slower_film_list AS
SELECT
    film.film_id AS fid
    ,film.title
    ,CAST(film.description AS NVARCHAR(MAX)) AS description
    ,category.name AS category
    ,film.rental_rate AS price
    ,film.length
    ,film.rating
    ,(
        SELECT
            CONCAT(
                UPPER(SUBSTRING(actor.first_name, 1, 1)),
                LOWER(SUBSTRING(actor.first_name, 2, 999))
            ) AS first_name
            ,CONCAT(
                UPPER(SUBSTRING(actor.last_name, 1, 1)),
                LOWER(SUBSTRING(actor.last_name, 2, 999))
            ) AS last_name
        FROM actor
        WHERE actor.actor_id = film_actor.actor_id
        FOR JSON AUTO
    ) AS actors
FROM
    category
    LEFT JOIN film_category ON film_category.category_id = category.category_id
    LEFT JOIN film ON film.film_id = film_category.film_id
    JOIN film_actor ON film_actor.film_id = film.film_id
GROUP BY
    film.film_id
    ,film.title
    ,CAST(film.description AS NVARCHAR(MAX))
    ,category.name
    ,film.rental_rate
    ,film.length
    ,film.rating
    ,film_actor.actor_id
;
