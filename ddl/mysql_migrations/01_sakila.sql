CREATE TABLE actor (
    actor_id INT AUTO_INCREMENT
    ,first_name VARCHAR(45) NOT NULL
    ,last_name VARCHAR(45) NOT NULL
    ,full_name VARCHAR(255) GENERATED ALWAYS AS (CONCAT(first_name, ' ', last_name)) VIRTUAL
    ,full_name_reversed VARCHAR(255) GENERATED ALWAYS AS (CONCAT(last_name, ' ', first_name)) STORED
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (actor_id)
    ,INDEX actor_last_name_idx (last_name)
);

CREATE TABLE category (
    category_id INT AUTO_INCREMENT
    ,name VARCHAR(45) NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (category_id)
);

CREATE TABLE country (
    country_id INT AUTO_INCREMENT
    ,country VARCHAR(50) NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (country_id)
);

CREATE TABLE city (
    city_id INT AUTO_INCREMENT
    ,city VARCHAR(50) NOT NULL
    ,country_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (city_id)
    ,INDEX city_country_id_idx (country_id)
);

CREATE TABLE address (
    address_id INT AUTO_INCREMENT
    ,address VARCHAR(50) NOT NULL
    ,address2 VARCHAR(50)
    ,district VARCHAR(20) NOT NULL
    ,city_id INT NOT NULL
    ,postal_code VARCHAR(10)
    ,phone VARCHAR(20) NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (address_id)
    ,INDEX address_city_id_idx (city_id)
);

CREATE TABLE language (
    language_id INT AUTO_INCREMENT
    ,name CHAR(20) NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (language_id)
);

CREATE TABLE film (
    film_id INT AUTO_INCREMENT
    ,title VARCHAR(255) NOT NULL
    ,description TEXT
    ,release_year INT
    ,language_id INT NOT NULL
    ,original_language_id INT
    ,rental_duration INT NOT NULL DEFAULT 3
    ,rental_rate DECIMAL(4,2) NOT NULL DEFAULT 4.99
    ,length INT
    ,replacement_cost DECIMAL(5,2) NOT NULL DEFAULT 19.99
    ,rating ENUM('G','PG','PG-13','R','NC-17') DEFAULT 'G'
    ,special_features JSON
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (film_id)
    ,CONSTRAINT film_year_check CHECK (release_year >= 1901 AND release_year <= 2155)
    ,INDEX film_title_idx (title)
    ,INDEX film_language_id_idx (language_id)
    ,INDEX film_original_language_id_idx (original_language_id)
);

CREATE TABLE film_actor (
    actor_id INT NOT NULL
    ,film_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,CONSTRAINT film_actor_actor_id_film_id_pkey PRIMARY KEY (actor_id, film_id)
    ,INDEX film_actor_film_id_idx (film_id)
);

CREATE TABLE film_category (
    film_id INT NOT NULL
    ,category_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,CONSTRAINT film_category_film_id_category_id_pkey PRIMARY KEY (film_id, category_id)
);

CREATE TABLE staff (
    staff_id INT AUTO_INCREMENT
    ,first_name VARCHAR(45) NOT NULL
    ,last_name VARCHAR(45) NOT NULL
    ,address_id INT NOT NULL
    ,picture MEDIUMBLOB
    ,email VARCHAR(50)
    ,store_id INT
    ,active BOOLEAN NOT NULL DEFAULT TRUE
    ,username VARCHAR(16) NOT NULL
    ,password VARCHAR(40)
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (staff_id)
    ,CONSTRAINT staff_email_key UNIQUE (email)
    ,INDEX staff_address_id_idx (address_id)
    ,INDEX staff_store_id_idx (store_id)
);

CREATE TABLE store (
    store_id INT AUTO_INCREMENT
    ,manager_staff_id INT NOT NULL
    ,address_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (store_id)
    ,INDEX store_manager_staff_id_idx (manager_staff_id)
    ,INDEX store_address_id_idx (address_id)
);

CREATE TABLE customer (
    customer_id INT AUTO_INCREMENT
    ,store_id INT NOT NULL
    ,first_name VARCHAR(45) NOT NULL
    ,last_name VARCHAR(45) NOT NULL
    ,email VARCHAR(50)
    ,address_id INT NOT NULL
    ,active BOOLEAN NOT NULL DEFAULT TRUE
    ,create_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (customer_id)
    ,CONSTRAINT customer_email_first_name_last_name_key UNIQUE (email, first_name, last_name)
    ,CONSTRAINT customer_email_key UNIQUE (email)
    ,INDEX customer_store_id_idx (store_id)
    ,INDEX customer_last_name_idx (last_name)
    ,INDEX customer_address_id_idx (address_id)
);

CREATE TABLE inventory (
    inventory_id INT AUTO_INCREMENT
    ,film_id INT NOT NULL
    ,store_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (inventory_id)
    ,INDEX inventory_store_id_film_id_idx (store_id, film_id)
    ,INDEX inventory_film_id_idx (film_id)
);

CREATE TABLE rental (
    rental_id INT AUTO_INCREMENT
    ,rental_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,inventory_id INT NOT NULL
    ,customer_id INT NOT NULL
    ,return_date DATETIME
    ,staff_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (rental_id)
    ,UNIQUE INDEX rental_inventory_id_customer_id_staff_id_idx (inventory_id, customer_id, staff_id)
    ,INDEX rental_inventory_id_idx (inventory_id)
    ,INDEX rental_customer_id_idx (customer_id)
    ,INDEX rental_staff_id_idx (staff_id)
);

CREATE TABLE payment (
    payment_id INT AUTO_INCREMENT
    ,customer_id INT NOT NULL
    ,staff_id INT NOT NULL
    ,rental_id INT
    ,amount DECIMAL(5,2) NOT NULL
    ,payment_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (payment_id)
    ,INDEX payment_customer_id_idx (customer_id)
    ,INDEX payment_staff_id_idx (staff_id)
    ,INDEX payment_rental_id_idx (rental_id)
);
