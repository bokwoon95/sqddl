CREATE OR ALTER FUNCTION inventory_held_by_customer(@p_inventory_id INT) RETURNS INT AS
BEGIN
    DECLARE @v_customer_id INT;

    SELECT @v_customer_id = customer_id
    FROM rental
    WHERE
        return_date IS NULL
        AND inventory_id = @p_inventory_id
    ;

    RETURN @v_customer_id;
END;
