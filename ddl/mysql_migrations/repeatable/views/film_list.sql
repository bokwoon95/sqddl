CREATE OR REPLACE VIEW film_list AS
SELECT
    film.film_id AS fid
    ,film.title
    ,film.description
    ,category.name AS category
    ,film.rental_rate AS price
    ,film.length
    ,film.rating
    ,json_arrayagg(CONCAT(actor.first_name, ' ', actor.last_name)) AS actors
FROM
    category
    LEFT JOIN film_category ON film_category.category_id = category.category_id
    LEFT JOIN film ON film.film_id = film_category.film_id
    JOIN film_actor ON film_actor.film_id = film.film_id
    JOIN actor ON actor.actor_id = film_actor.actor_id
GROUP BY
    film.film_id
    ,film.title
    ,film.description
    ,category.name
    ,film.rental_rate
    ,film.length
    ,film.rating
;
