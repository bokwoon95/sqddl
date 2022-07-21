CREATE OR ALTER PROCEDURE film_in_stock(@p_film_id INT, @p_store_id INT, @p_film_count INT OUTPUT) AS
BEGIN
    SELECT @p_film_count = COUNT(*)
    FROM inventory
    WHERE
        film_id = @p_film_id
        AND store_id = @p_store_id
        AND dbo.inventory_in_stock(inventory_id) = 1
    ;
END;
