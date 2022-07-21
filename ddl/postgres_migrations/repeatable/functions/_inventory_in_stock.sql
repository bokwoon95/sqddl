CREATE OR REPLACE FUNCTION inventory_in_stock(p_inventory_id INT) RETURNS BOOLEAN AS $$ DECLARE
    v_rentals INT;
    v_out     INT;
BEGIN
    -- AN ITEM IS IN-STOCK IF THERE ARE EITHER NO ROWS IN THE rental TABLE
    -- FOR THE ITEM OR ALL ROWS HAVE return_date POPULATED
    SELECT COUNT(*) INTO v_rentals
    FROM rental
    WHERE inventory_id = p_inventory_id
    ;
    IF v_rentals = 0 THEN
      RETURN TRUE;
    END IF;
    SELECT
        COUNT(rental_id) INTO v_out
    FROM
        inventory
        LEFT JOIN rental USING(inventory_id)
    WHERE
        inventory.inventory_id = p_inventory_id
        AND rental.return_date IS NULL
    ;
    IF v_out > 0 THEN
      RETURN FALSE;
    ELSE
      RETURN TRUE;
    END IF;
END $$ LANGUAGE plpgsql;
