CREATE OR ALTER VIEW full_address WITH SCHEMABINDING AS
SELECT
    country.country_id
    ,city.city_id
    ,address.address_id
    ,country.country
    ,city.city
    ,address.address
    ,address.address2
    ,address.district
    ,address.postal_code
    ,address.phone
    ,address.last_update
FROM
    dbo.address
    JOIN dbo.city ON city.city_id = address.city_id
    JOIN dbo.country ON country.country_id = city.country_id
;
