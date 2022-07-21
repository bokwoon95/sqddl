CREATE OR ALTER FUNCTION get_customer_balance(@p_customer_id INT, @p_effective_date DATETIME) RETURNS DECIMAL(5,2) AS
BEGIN
    -- OK, WE NEED TO CALCULATE THE CURRENT BALANCE GIVEN A CUSTOMER_ID AND A DATE
    -- THAT WE WANT THE BALANCE TO BE EFFECTIVE FOR. THE BALANCE IS:
    --    1) RENTAL FEES FOR ALL PREVIOUS RENTALS
    --    2) ONE DOLLAR FOR EVERY DAY THE PREVIOUS RENTALS ARE OVERDUE
    --    3) IF A FILM IS MORE THAN RENTAL_DURATION * 2 OVERDUE, CHARGE THE REPLACEMENT_COST
    --    4) SUBTRACT ALL PAYMENTS MADE BEFORE THE DATE SPECIFIED
    DECLARE @v_rentfees DECIMAL(5,2); -- FEES PAID TO RENT THE VIDEOS INITIALLY
    DECLARE @v_overfees INT;          -- LATE FEES FOR PRIOR RENTALS
    DECLARE @v_payments DECIMAL(5,2); -- SUM OF PAYMENTS MADE PREVIOUSLY

    SELECT @v_rentfees = COALESCE(SUM(film.rental_rate),0)
    FROM film, inventory, rental
    WHERE
        film.film_id = inventory.film_id
        AND inventory.inventory_id = rental.inventory_id
        AND rental.rental_date <= @p_effective_date
        AND rental.customer_id = @p_customer_id
    ;

    SELECT @v_overfees = COALESCE(SUM(IIF(
        DATEDIFF(DAY, rental.return_date, rental.rental_date) > film.rental_duration,
        DATEDIFF(DAY, rental.return_date, rental.rental_date) - film.rental_duration,
        0
    )), 0)
    FROM rental, inventory, film
    WHERE
        film.film_id = inventory.film_id
        AND inventory.inventory_id = rental.inventory_id
        AND rental.rental_date <= @p_effective_date
        AND rental.customer_id = @p_customer_id
    ;

    SELECT @v_payments = COALESCE(SUM(payment.amount),0)
    FROM payment
    WHERE
        payment.payment_date <= @p_effective_date
        AND payment.customer_id = @p_customer_id
    ;

    RETURN @v_rentfees + @v_overfees - @v_payments;
END;
