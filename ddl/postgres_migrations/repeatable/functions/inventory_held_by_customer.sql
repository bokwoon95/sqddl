CREATE OR REPLACE FUNCTION inventory_held_by_customer(p_inventory_id INT) RETURNS INT AS $$ DECLARE
    v_customer_id INT;
BEGIN
    SELECT customer_id
    INTO v_customer_id
    FROM rental
    WHERE
        return_date IS NULL
        AND inventory_id = p_inventory_id
    ;

    RETURN v_customer_id;
END $$ LANGUAGE plpgsql;
