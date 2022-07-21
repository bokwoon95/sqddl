CREATE OR REPLACE PROCEDURE film_in_stock(p_film_id INT, p_store_id INT, OUT p_film_count INT) AS $$
BEGIN
    SELECT COUNT(*)
    INTO p_film_count
    FROM inventory
    WHERE
        film_id = p_film_id
        AND store_id = p_store_id
        AND inventory_in_stock(inventory_id)
    ;
END
$$ LANGUAGE plpgsql;
