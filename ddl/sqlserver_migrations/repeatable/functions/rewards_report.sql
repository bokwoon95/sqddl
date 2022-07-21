CREATE OR ALTER PROCEDURE rewards_report (
    @min_monthly_purchases INT
    ,@min_dollar_amount_purchased DECIMAL(10,2)
    ,@count_rewardees INT OUTPUT
) AS
BEGIN
    DECLARE @last_month_start DATE;
    DECLARE @last_month_end DATE;

    /* Some sanity checks... */
    IF @min_monthly_purchases = 0
        THROW 50001, 'Minimum monthly purchases parameter must be > 0', 1;
    IF @min_dollar_amount_purchased = 0.00
        THROW 50001, 'Minimum monthly dollar amount purchased parameter must be > $0.00', 1;

    /* Determine start and end time periods */
    SELECT @last_month_start = DATEADD(MONTH, 1, CURRENT_TIMESTAMP);
    SELECT @last_month_start = CONVERT(DATE, CONCAT(YEAR(@last_month_start),'-',MONTH(@last_month_start),'-01'));
    SELECT @last_month_end = EOMONTH(@last_month_start);

    /*
    Create a temporary storage area for
    Customer IDs.
    */
    CREATE TABLE ##tmpCustomer (customer_id INT NOT NULL PRIMARY KEY);

    /*
    Find all customers meeting the
    monthly purchase requirements
    */
    INSERT INTO ##tmpCustomer (customer_id)
    SELECT p.customer_id
    FROM payment AS p
    WHERE CONVERT(DATE, p.payment_date) BETWEEN @last_month_start AND @last_month_end
    GROUP BY customer_id
    HAVING SUM(p.amount) > @min_dollar_amount_purchased
    AND COUNT(customer_id) > @min_monthly_purchases;

    /* Populate OUT parameter with count of found customers */
    SELECT @count_rewardees = COUNT(*) FROM ##tmpCustomer;

    /*
    Output ALL customer information of matching rewardees.
    Customize output as needed.
    */
    SELECT c.*
    FROM ##tmpCustomer AS t
    INNER JOIN customer AS c ON t.customer_id = c.customer_id;

    /* Clean up */
    DROP TABLE ##tmpCustomer;
END;
