DROP FUNCTION IF EXISTS inventory_held_by_customer;
CREATE FUNCTION inventory_held_by_customer(p_inventory_id INT) RETURNS INT
READS SQL DATA
BEGIN
    DECLARE v_customer_id INT;
    DECLARE EXIT HANDLER FOR NOT FOUND RETURN NULL;

    SELECT customer_id
    INTO v_customer_id
    FROM rental
    WHERE
        return_date IS NULL
        AND inventory_id = p_inventory_id
    ;

    RETURN v_customer_id;
END;
