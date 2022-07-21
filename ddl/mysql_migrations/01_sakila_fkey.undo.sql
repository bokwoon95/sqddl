ALTER TABLE city
    DROP CONSTRAINT city_country_id_fkey
;

ALTER TABLE address
    DROP CONSTRAINT address_city_id_fkey
;

ALTER TABLE film
    DROP CONSTRAINT film_language_id_fkey
    ,DROP CONSTRAINT film_original_language_id_fkey
;

ALTER TABLE film_actor
    DROP CONSTRAINT film_actor_film_id_fkey
    ,DROP CONSTRAINT film_actor_actor_id_fkey
;

ALTER TABLE film_category
    DROP CONSTRAINT film_category_film_id_fkey
    ,DROP CONSTRAINT film_category_category_id_fkey
;

ALTER TABLE staff
    DROP CONSTRAINT staff_address_id_fkey
    ,DROP CONSTRAINT staff_store_id_fkey
;

ALTER TABLE store
    DROP CONSTRAINT store_manager_staff_id_fkey
    ,DROP CONSTRAINT store_address_id_fkey
;

ALTER TABLE customer
    DROP CONSTRAINT customer_address_id_fkey
    ,DROP CONSTRAINT customer_store_id_fkey
;

ALTER TABLE inventory
    DROP CONSTRAINT inventory_film_id_fkey
    ,DROP CONSTRAINT inventory_store_id_fkey
;

ALTER TABLE rental
    DROP CONSTRAINT rental_inventory_id_fkey
    ,DROP CONSTRAINT rental_customer_id_fkey
    ,DROP CONSTRAINT rental_staff_id_fkey
;

ALTER TABLE payment
    DROP CONSTRAINT payment_customer_id_fkey
    ,DROP CONSTRAINT payment_staff_id_fkey
    ,DROP CONSTRAINT payment_rental_id_fkey
;
