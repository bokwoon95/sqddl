CREATE TABLE actor (
    actor_id INT IDENTITY
    ,first_name NVARCHAR(45) NOT NULL
    ,last_name NVARCHAR(45) NOT NULL
    ,full_name AS CONCAT(first_name, ' ', last_name)
    ,full_name_reversed AS CONCAT(last_name, ' ', first_name) PERSISTED
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT actor_actor_id_pkey PRIMARY KEY (actor_id)
);

CREATE INDEX actor_last_name_idx ON actor (last_name);

CREATE TABLE category (
    category_id INT IDENTITY
    ,name NVARCHAR(45) NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT category_category_id_pkey PRIMARY KEY (category_id)
);

CREATE TABLE country (
    country_id INT IDENTITY
    ,country NVARCHAR(50) NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT country_country_id_pkey PRIMARY KEY (country_id)
);

CREATE TABLE city (
    city_id INT IDENTITY
    ,city NVARCHAR(50) NOT NULL
    ,country_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT city_city_id_pkey PRIMARY KEY (city_id)
);

CREATE INDEX city_country_id_idx ON city (country_id);

CREATE TABLE address (
    address_id INT IDENTITY
    ,address NVARCHAR(50) NOT NULL
    ,address2 NVARCHAR(50)
    ,district NVARCHAR(20) NOT NULL
    ,city_id INT NOT NULL
    ,postal_code NVARCHAR(10)
    ,phone VARCHAR(20) NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT address_address_id_pkey PRIMARY KEY (address_id)
);

CREATE INDEX address_city_id_idx on address (city_id);

CREATE TABLE language (
    language_id INT IDENTITY
    ,name CHAR(20) NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT language_language_id_pkey PRIMARY KEY (language_id)
);

CREATE TABLE film (
    film_id INT IDENTITY
    ,title NVARCHAR(255) NOT NULL
    ,description TEXT
    ,release_year INT
    ,language_id INT NOT NULL
    ,original_language_id INT
    ,rental_duration INT NOT NULL DEFAULT 3
    ,rental_rate DECIMAL(4,2) NOT NULL DEFAULT 4.99
    ,length INT
    ,replacement_cost DECIMAL(5,2) NOT NULL DEFAULT 19.99
    ,rating NVARCHAR(255) DEFAULT 'G'
    ,special_features NVARCHAR(MAX)
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT film_film_id_pkey PRIMARY KEY (film_id)
    ,CONSTRAINT film_year_check CHECK (release_year >= 1901 AND release_year <= 2155)
    ,CONSTRAINT film_rating_check CHECK (rating IN ('G','PG','PG-13','R','NC-17'))
);

CREATE INDEX film_title_idx ON film (title);

CREATE INDEX film_language_id_idx ON film (language_id);

CREATE INDEX film_original_language_id_idx ON film (original_language_id);

CREATE TABLE film_actor (
    actor_id INT NOT NULL
    ,film_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT film_actor_actor_id_film_id_pkey PRIMARY KEY (actor_id, film_id)
);

CREATE INDEX film_actor_film_id_idx ON film_actor (film_id);

CREATE TABLE film_category (
    film_id INT NOT NULL
    ,category_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT film_category_film_id_category_id_pkey PRIMARY KEY (film_id, category_id)
);

CREATE TABLE staff (
    staff_id INT IDENTITY
    ,first_name NVARCHAR(45) NOT NULL
    ,last_name NVARCHAR(45) NOT NULL
    ,address_id INT NOT NULL
    ,picture VARBINARY(MAX)
    ,email NVARCHAR(50)
    ,store_id INT
    ,active BIT NOT NULL DEFAULT 1
    ,username NVARCHAR(16) NOT NULL
    ,password NVARCHAR(40)
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT staff_staff_id_pkey PRIMARY KEY (staff_id)
    ,CONSTRAINT staff_email_key UNIQUE (email)
);

CREATE INDEX staff_address_id_idx ON staff (address_id);

CREATE INDEX staff_store_id_idx ON staff (store_id);

CREATE TABLE store (
    store_id INT IDENTITY
    ,manager_staff_id INT NOT NULL
    ,address_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT store_store_id_pkey PRIMARY KEY (store_id)
);

CREATE INDEX store_manager_staff_id_idx ON store (manager_staff_id);

CREATE INDEX store_address_id_idx ON store (address_id);

CREATE TABLE customer (
    customer_id INT IDENTITY
    ,store_id INT NOT NULL
    ,first_name NVARCHAR(45) NOT NULL
    ,last_name NVARCHAR(45) NOT NULL
    ,email NVARCHAR(50)
    ,address_id INT NOT NULL
    ,active BIT NOT NULL DEFAULT 1
    ,create_date DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT customer_customer_id_pkey PRIMARY KEY (customer_id)
    ,CONSTRAINT customer_email_first_name_last_name_key UNIQUE (email, first_name, last_name)
    ,CONSTRAINT customer_email_key UNIQUE (email)
);

CREATE INDEX customer_store_id_idx ON customer (store_id);

CREATE INDEX customer_last_name_idx ON customer (last_name);

CREATE INDEX customer_address_id_idx ON customer (address_id);

CREATE TABLE inventory (
    inventory_id INT IDENTITY
    ,film_id INT NOT NULL
    ,store_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT inventory_inventory_id_pkey PRIMARY KEY (inventory_id)
);

CREATE INDEX inventory_store_id_film_id_idx ON inventory (store_id, film_id);

CREATE INDEX inventory_film_id_idx ON inventory (film_id);

CREATE TABLE rental (
    rental_id INT IDENTITY
    ,rental_date DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,inventory_id INT NOT NULL
    ,customer_id INT NOT NULL
    ,return_date DATETIMEOFFSET
    ,staff_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT rental_rental_id_pkey PRIMARY KEY (rental_id)
);

CREATE UNIQUE INDEX rental_inventory_id_customer_id_staff_id_idx ON rental (inventory_id, customer_id, staff_id);

CREATE INDEX rental_inventory_id_idx ON rental (inventory_id);

CREATE INDEX rental_customer_id_idx ON rental (customer_id);

CREATE INDEX rental_staff_id_idx ON rental (staff_id);

CREATE TABLE payment (
    payment_id INT IDENTITY
    ,customer_id INT NOT NULL
    ,staff_id INT NOT NULL
    ,rental_id INT
    ,amount DECIMAL(5,2) NOT NULL
    ,payment_date DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT CURRENT_TIMESTAMP

    ,CONSTRAINT payment_payment_id_pkey PRIMARY KEY (payment_id)
);

CREATE INDEX payment_customer_id_idx ON payment (customer_id);

CREATE INDEX payment_staff_id_idx ON payment (staff_id);

CREATE INDEX payment_rental_id_idx ON payment (rental_id);

ALTER TABLE city ADD CONSTRAINT city_country_id_fkey FOREIGN KEY (country_id) REFERENCES country (country_id) ON UPDATE CASCADE ON DELETE NO ACTION;

ALTER TABLE address ADD CONSTRAINT address_city_id_fkey FOREIGN KEY (city_id) REFERENCES city (city_id) ON UPDATE CASCADE ON DELETE NO ACTION;

ALTER TABLE film ADD CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id);
ALTER TABLE film ADD CONSTRAINT film_original_language_id_fkey FOREIGN KEY (original_language_id) REFERENCES language (language_id);

ALTER TABLE film_actor ADD CONSTRAINT film_actor_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE NO ACTION;
ALTER TABLE film_actor ADD CONSTRAINT film_actor_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES actor (actor_id) ON UPDATE CASCADE ON DELETE NO ACTION;

ALTER TABLE film_category ADD CONSTRAINT film_category_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE NO ACTION;
ALTER TABLE film_category ADD CONSTRAINT film_category_category_id_fkey FOREIGN KEY (category_id) REFERENCES category (category_id) ON UPDATE CASCADE ON DELETE NO ACTION;

ALTER TABLE staff ADD CONSTRAINT staff_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE NO ACTION;
-- ALTER TABLE staff ADD CONSTRAINT staff_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE NO ACTION; /* store already depends on staff; don't create a circular dependency */

ALTER TABLE store ADD CONSTRAINT store_manager_staff_id_fkey FOREIGN KEY (manager_staff_id) REFERENCES staff (staff_id);
ALTER TABLE store ADD CONSTRAINT store_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id);

ALTER TABLE customer ADD CONSTRAINT customer_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE NO ACTION;
ALTER TABLE customer ADD CONSTRAINT customer_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE NO ACTION;

ALTER TABLE inventory ADD CONSTRAINT inventory_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE NO ACTION;
ALTER TABLE inventory ADD CONSTRAINT inventory_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE NO ACTION;

ALTER TABLE rental ADD CONSTRAINT rental_inventory_id_fkey FOREIGN KEY (inventory_id) REFERENCES inventory (inventory_id);
ALTER TABLE rental ADD CONSTRAINT rental_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer (customer_id);
ALTER TABLE rental ADD CONSTRAINT rental_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff (staff_id);

ALTER TABLE payment ADD CONSTRAINT payment_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer (customer_id);
ALTER TABLE payment ADD CONSTRAINT payment_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff (staff_id);
ALTER TABLE payment ADD CONSTRAINT payment_rental_id_fkey FOREIGN KEY (rental_id) REFERENCES rental (rental_id) ON UPDATE CASCADE ON DELETE SET NULL;
