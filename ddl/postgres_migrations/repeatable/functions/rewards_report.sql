CREATE OR REPLACE FUNCTION rewards_report (min_monthly_purchases INT, min_dollar_amount_purchased DECIMAL(10,2)) RETURNS SETOF customer
AS $$ DECLARE
    last_month_start DATE;
    last_month_end DATE;
    rr RECORD;
    -- tmpSQL TEXT;
BEGIN
    /* Some sanity checks... */
    IF min_monthly_purchases = 0 THEN
        RAISE EXCEPTION 'Minimum monthly purchases parameter must be > 0';
    END IF;
    IF min_dollar_amount_purchased = 0.00 THEN
        RAISE EXCEPTION 'Minimum monthly dollar amount purchased parameter must be > $0.00';
    END IF;

    last_month_start := CURRENT_DATE - '1 month'::INTERVAL;
    last_month_start := to_date((extract(YEAR FROM last_month_start) || '-' || extract(MONTH FROM last_month_start) || '-01'),'YYYY-MM-DD');
    last_month_end := LAST_DAY(last_month_start);

    /*
    Create a temporary storage area for Customer IDs.
    */
    CREATE TEMPORARY TABLE tmpCustomer (customer_id INT NOT NULL PRIMARY KEY);

    /*
    Find all customers meeting the monthly purchase requirements
    */
    INSERT INTO tmpCustomer (customer_id)
    SELECT p.customer_id
    FROM payment AS p
    WHERE DATE(p.payment_date) BETWEEN last_month_start AND last_month_end
    GROUP BY customer_id
    HAVING SUM(p.amount) > min_dollar_amount_purchased
    AND COUNT(customer_id) > min_monthly_purchases;

    /*
    Output ALL customer information of matching rewardees.
    Customize output as needed.
    */
    FOR rr IN SELECT c.* FROM tmpCustomer AS t INNER JOIN customer AS c ON t.customer_id = c.customer_id
    LOOP
        RETURN NEXT rr;
    END LOOP;

    /* Clean up */
    DROP TABLE tmpCustomer;
END
$$ LANGUAGE plpgsql SECURITY DEFINER;
