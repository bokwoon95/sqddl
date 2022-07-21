CREATE OR REPLACE VIEW film_list AS
SELECT
    film.film_id AS fid
    ,film.title
    ,film.description
    ,category.name AS category
    ,film.rental_rate AS price
    ,film.length
    ,film.rating
    ,jsonb_agg(actor.first_name || ' ' || actor.last_name) AS actors
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

CREATE OR REPLACE FUNCTION update_film_list() RETURNS TRIGGER AS $$ BEGIN
    IF TG_OP <> 'UPDATE' THEN
        RAISE EXCEPTION 'Invalid operation on film_list: %', TG_OP;
    END IF;

    IF OLD.fid IS NULL THEN
        RAISE EXCEPTION 'Unable to update film_list, film_list.fid is missing';
    END IF;

    IF NEW.fid <> OLD.fid THEN
        RAISE EXCEPTION 'You are not allowed to update film_list.fid';
    END IF;

    IF NEW.category <> OLD.category THEN
        RAISE EXCEPTION 'You are not allowed to update film_list.category';
    END IF;

    IF NEW.actors <> OLD.actors THEN
        RAISE EXCEPTION 'You are not allowed to update film_list.actors';
    END IF;

    IF NEW.title <> OLD.title
        OR NEW.description <> OLD.description
        OR NEW.price <> OLD.price
        OR NEW.length <> OLD.length
        OR NEW.rating <> OLD.rating
    THEN
        UPDATE film
        SET
            title = NEW.title
            ,description = NEW.description
            ,rental_rate = NEW.price
            ,length = NEW.length
            ,rating = NEW.rating
        WHERE
            film_id = NEW.fid
        ;
    END IF;

    RETURN NEW;
END; $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_film_list_trg ON film_list;
CREATE TRIGGER update_film_list_trg INSTEAD OF UPDATE ON film_list
FOR EACH ROW EXECUTE PROCEDURE update_film_list();
