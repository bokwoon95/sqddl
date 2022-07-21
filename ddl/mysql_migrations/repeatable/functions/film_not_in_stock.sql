DROP PROCEDURE IF EXISTS film_not_in_stock;
CREATE PROCEDURE film_not_in_stock(IN p_film_id INT, IN p_store_id INT, OUT p_film_count INT)
READS SQL DATA
BEGIN
    SELECT COUNT(*)
    INTO p_film_count
    FROM inventory
    WHERE
        film_id = p_film_id
        AND store_id = p_store_id
        AND NOT inventory_in_stock(inventory_id)
    ;
END;
