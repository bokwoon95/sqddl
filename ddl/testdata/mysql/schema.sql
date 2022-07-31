CREATE SCHEMA IF NOT EXISTS sakila;

CREATE TABLE actor (
    actor_id INT NOT NULL AUTO_INCREMENT
    ,first_name VARCHAR(45) NOT NULL
    ,last_name VARCHAR(45) NOT NULL
    ,full_name VARCHAR(255) AS (concat(`first_name`,_utf8mb4' ',`last_name`))
    ,full_name_reversed VARCHAR(255) AS (concat(`last_name`,_utf8mb4' ',`first_name`)) STORED
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (actor_id)
);

CREATE TABLE address (
    address_id INT NOT NULL AUTO_INCREMENT
    ,address VARCHAR(50) NOT NULL
    ,address2 VARCHAR(50)
    ,district VARCHAR(20) NOT NULL
    ,city_id INT NOT NULL
    ,postal_code VARCHAR(10)
    ,phone VARCHAR(20) NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (address_id)
);

CREATE TABLE category (
    category_id INT NOT NULL AUTO_INCREMENT
    ,name VARCHAR(45) NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (category_id)
);

CREATE TABLE city (
    city_id INT NOT NULL AUTO_INCREMENT
    ,city VARCHAR(50) NOT NULL
    ,country_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (city_id)
);

CREATE TABLE country (
    country_id INT NOT NULL AUTO_INCREMENT
    ,country VARCHAR(50) NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (country_id)
);

CREATE TABLE customer (
    customer_id INT NOT NULL AUTO_INCREMENT
    ,store_id INT NOT NULL
    ,first_name VARCHAR(45) NOT NULL
    ,last_name VARCHAR(45) NOT NULL
    ,email VARCHAR(50)
    ,address_id INT NOT NULL
    ,active TINYINT(1) NOT NULL DEFAULT 1
    ,create_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (customer_id)
);

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

CREATE TABLE employee_department (
    employee_id BINARY(16) NOT NULL
    ,department_id BINARY(16) NOT NULL

    ,PRIMARY KEY (employee_id, department_id)
);

CREATE TABLE film (
    film_id INT NOT NULL AUTO_INCREMENT
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
);

CREATE TABLE film_actor (
    actor_id INT NOT NULL
    ,film_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (actor_id, film_id)
);

CREATE TABLE film_category (
    film_id INT NOT NULL
    ,category_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

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
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (inventory_id)
);

CREATE TABLE language (
    language_id INT NOT NULL AUTO_INCREMENT
    ,name CHAR(20) NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (language_id)
);

CREATE TABLE payment (
    payment_id INT NOT NULL AUTO_INCREMENT
    ,customer_id INT NOT NULL
    ,staff_id INT NOT NULL
    ,rental_id INT
    ,amount DECIMAL(5,2) NOT NULL
    ,payment_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (payment_id)
);

CREATE TABLE rental (
    rental_id INT NOT NULL AUTO_INCREMENT
    ,rental_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    ,inventory_id INT NOT NULL
    ,customer_id INT NOT NULL
    ,return_date DATETIME
    ,staff_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (rental_id)
);

CREATE TABLE staff (
    staff_id INT NOT NULL AUTO_INCREMENT
    ,first_name VARCHAR(45) NOT NULL
    ,last_name VARCHAR(45) NOT NULL
    ,address_id INT NOT NULL
    ,picture MEDIUMBLOB
    ,email VARCHAR(50)
    ,store_id INT
    ,active TINYINT(1) NOT NULL DEFAULT 1
    ,username VARCHAR(16) NOT NULL
    ,password VARCHAR(40)
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (staff_id)
);

CREATE TABLE store (
    store_id INT NOT NULL AUTO_INCREMENT
    ,manager_staff_id INT NOT NULL
    ,address_id INT NOT NULL
    ,last_update DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP

    ,PRIMARY KEY (store_id)
);

CREATE TABLE task (
    task_id BINARY(16) NOT NULL
    ,employee_id BINARY(16) NOT NULL
    ,department_id BINARY(16) NOT NULL
    ,task VARCHAR(255) NOT NULL
    ,data JSON

    ,PRIMARY KEY (task_id)
);
