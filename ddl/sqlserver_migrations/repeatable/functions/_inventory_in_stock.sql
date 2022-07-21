CREATE OR ALTER FUNCTION inventory_in_stock(@p_inventory_id INT) RETURNS BIT AS
BEGIN
    DECLARE @v_rentals INT;
    DECLARE @v_out     INT;

    -- AN ITEM IS IN-STOCK IF THERE ARE EITHER NO ROWS IN THE rental TABLE
    -- FOR THE ITEM OR ALL ROWS HAVE return_date POPULATED

    SELECT @v_rentals = COUNT(*)
    FROM rental
    WHERE inventory_id = @p_inventory_id
    ;

    IF @v_rentals = 0 RETURN 1;

    SELECT @v_out = COUNT(rental_id)
    FROM
        inventory
        LEFT JOIN rental ON rental.inventory_id = inventory.inventory_id
    WHERE
        inventory.inventory_id = @p_inventory_id
        AND rental.return_date IS NULL
    ;

    IF @v_out > 0 RETURN 0;

    RETURN 1;
END;
