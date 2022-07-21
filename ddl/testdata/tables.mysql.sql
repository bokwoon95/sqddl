CREATE TABLE actor (
    actor_id INT NOT NULL AUTO_INCREMENT
    ,first_name VARCHAR(45) NOT NULL
    ,last_name VARCHAR(45) NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (actor_id)
);

CREATE INDEX actor_last_name_idx ON actor (last_name);

CREATE TABLE address (
    address_id INT NOT NULL AUTO_INCREMENT
    ,address VARCHAR(50) NOT NULL
    ,address2 VARCHAR(50)
    ,district VARCHAR(20) NOT NULL
    ,city_id INT NOT NULL
    ,postal_code VARCHAR(10)
    ,phone VARCHAR(20) NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (address_id)
);

CREATE INDEX address_city_id_idx ON address (city_id);

CREATE TABLE category (
    category_id INT NOT NULL AUTO_INCREMENT
    ,name VARCHAR(45) NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (category_id)
);

CREATE TABLE city (
    city_id INT NOT NULL AUTO_INCREMENT
    ,city VARCHAR(50) NOT NULL
    ,country_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (city_id)
);

CREATE INDEX city_country_id_idx ON city (country_id);

CREATE TABLE country (
    country_id INT NOT NULL AUTO_INCREMENT
    ,country VARCHAR(50) NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (country_id)
);

CREATE TABLE customer (
    customer_id INT NOT NULL AUTO_INCREMENT
    ,store_id INT NOT NULL
    ,first_name VARCHAR(45) NOT NULL
    ,last_name VARCHAR(45) NOT NULL
    ,email VARCHAR(50)
    ,address_id INT NOT NULL
    ,active INT NOT NULL DEFAULT TRUE
    ,create_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (customer_id)
    ,CONSTRAINT customer_email_key UNIQUE (email)
);

CREATE INDEX customer_store_id_idx ON customer (store_id);

CREATE INDEX customer_last_name_idx ON customer (last_name);

CREATE INDEX customer_address_id_idx ON customer (address_id);

CREATE TABLE department (
    department_id BINARY(16) NOT NULL
    ,name VARCHAR(255) NOT NULL

    ,PRIMARY KEY (department_id)
);

CREATE TABLE employee (
    employee_id BINARY(16) NOT NULL
    ,name VARCHAR(255) NOT NULL
    ,title VARCHAR(255) NOT NULL
    ,manager_id BINARY(16)

    ,PRIMARY KEY (employee_id)
);

CREATE INDEX employee_manager_id_idx ON employee (manager_id);

CREATE TABLE employee_department (
    employee_id BINARY(16) NOT NULL
    ,department_id BINARY(16) NOT NULL

    ,PRIMARY KEY (employee_id, department_id)
);

CREATE INDEX employee_department_employee_id_idx ON employee_department (employee_id);

CREATE INDEX employee_department_department_id_idx ON employee_department (department_id);

CREATE TABLE film (
    film_id INT NOT NULL AUTO_INCREMENT
    ,title VARCHAR(255) NOT NULL
    ,description VARCHAR(255)
    ,release_year INT
    ,language_id INT NOT NULL
    ,original_language_id INT
    ,rental_duration INT NOT NULL DEFAULT 3
    ,rental_rate DECIMAL(4,2) NOT NULL DEFAULT 4.99
    ,length INT
    ,replacement_cost DECIMAL(5,2) NOT NULL DEFAULT 19.99
    ,rating VARCHAR(255) DEFAULT 'G'
    ,special_features JSON
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (film_id)
);

CREATE INDEX film_title_idx ON film (title);

CREATE INDEX film_language_id_idx ON film (language_id);

CREATE INDEX film_original_language_id_idx ON film (original_language_id);

CREATE TABLE film_actor (
    actor_id INT NOT NULL
    ,film_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (actor_id, film_id)
);

CREATE INDEX film_actor_film_id_idx ON film_actor (film_id);

CREATE TABLE film_category (
    film_id INT NOT NULL
    ,category_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (film_id, category_id)
);

CREATE TABLE film_text (
    film_id INT NOT NULL
    ,title VARCHAR(255)
    ,description TEXT

    ,PRIMARY KEY (film_id)
);

CREATE TABLE inventory (
    inventory_id INT NOT NULL AUTO_INCREMENT
    ,film_id INT NOT NULL
    ,store_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (inventory_id)
);

CREATE INDEX inventory_film_id_idx ON inventory (film_id);

CREATE TABLE language (
    language_id INT NOT NULL AUTO_INCREMENT
    ,name VARCHAR(20) NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (language_id)
);

CREATE TABLE payment (
    payment_id INT NOT NULL AUTO_INCREMENT
    ,customer_id INT NOT NULL
    ,staff_id INT NOT NULL
    ,rental_id INT
    ,amount REAL NOT NULL
    ,payment_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (payment_id)
);

CREATE INDEX payment_customer_id_idx ON payment (customer_id);

CREATE INDEX payment_staff_id_idx ON payment (staff_id);

CREATE INDEX payment_rental_id_idx ON payment (rental_id);

CREATE TABLE rental (
    rental_id INT NOT NULL AUTO_INCREMENT
    ,rental_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    ,inventory_id INT NOT NULL
    ,customer_id INT NOT NULL
    ,return_date TIMESTAMP
    ,staff_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (rental_id)
);

CREATE INDEX rental_inventory_id_idx ON rental (inventory_id);

CREATE INDEX rental_customer_id_idx ON rental (customer_id);

CREATE INDEX rental_staff_id_idx ON rental (staff_id);

CREATE TABLE staff (
    staff_id INT NOT NULL AUTO_INCREMENT
    ,first_name VARCHAR(255) NOT NULL
    ,last_name VARCHAR(255) NOT NULL
    ,address_id INT NOT NULL
    ,picture MEDIUMBLOB
    ,email VARCHAR(255)
    ,store_id INT
    ,active INT NOT NULL DEFAULT TRUE
    ,username VARCHAR(255) NOT NULL
    ,password VARCHAR(255)
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (staff_id)
    ,CONSTRAINT staff_email_key UNIQUE (email)
);

CREATE INDEX staff_address_id_idx ON staff (address_id);

CREATE INDEX staff_store_id_idx ON staff (store_id);

CREATE TABLE store (
    store_id INT NOT NULL AUTO_INCREMENT
    ,manager_staff_id INT NOT NULL
    ,address_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (store_id)
);

CREATE INDEX store_manager_staff_id_idx ON store (manager_staff_id);

CREATE INDEX store_address_id_idx ON store (address_id);

CREATE TABLE task (
    task_id BINARY(16) NOT NULL
    ,employee_id BINARY(16) NOT NULL
    ,department_id BINARY(16) NOT NULL
    ,task VARCHAR(255) NOT NULL
    ,data JSON

    ,PRIMARY KEY (task_id)
);

ALTER TABLE address ADD CONSTRAINT address_city_id_fkey FOREIGN KEY (city_id) REFERENCES city (city_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE city ADD CONSTRAINT city_country_id_fkey FOREIGN KEY (country_id) REFERENCES country (country_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE customer ADD CONSTRAINT customer_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE customer ADD CONSTRAINT customer_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE employee ADD CONSTRAINT employee_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES employee (employee_id);

ALTER TABLE employee_department ADD CONSTRAINT employee_department_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES employee (employee_id);

ALTER TABLE employee_department ADD CONSTRAINT employee_department_department_id_fkey FOREIGN KEY (department_id) REFERENCES department (department_id);

ALTER TABLE film ADD CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE film ADD CONSTRAINT film_original_language_id_fkey FOREIGN KEY (original_language_id) REFERENCES language (language_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE film_actor ADD CONSTRAINT film_actor_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES actor (actor_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE film_actor ADD CONSTRAINT film_actor_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE film_category ADD CONSTRAINT film_category_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE film_category ADD CONSTRAINT film_category_category_id_fkey FOREIGN KEY (category_id) REFERENCES category (category_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE inventory ADD CONSTRAINT inventory_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE inventory ADD CONSTRAINT inventory_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE payment ADD CONSTRAINT payment_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer (customer_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE payment ADD CONSTRAINT payment_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff (staff_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE payment ADD CONSTRAINT payment_rental_id_fkey FOREIGN KEY (rental_id) REFERENCES rental (rental_id) ON UPDATE CASCADE ON DELETE SET NULL;

ALTER TABLE rental ADD CONSTRAINT rental_inventory_id_fkey FOREIGN KEY (inventory_id) REFERENCES inventory (inventory_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE rental ADD CONSTRAINT rental_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer (customer_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE rental ADD CONSTRAINT rental_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff (staff_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE staff ADD CONSTRAINT staff_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE staff ADD CONSTRAINT staff_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE store ADD CONSTRAINT store_manager_staff_id_fkey FOREIGN KEY (manager_staff_id) REFERENCES staff (staff_id) ON UPDATE CASCADE ON DELETE RESTRICT;

ALTER TABLE store ADD CONSTRAINT store_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE RESTRICT;
