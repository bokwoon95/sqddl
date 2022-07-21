IF SCHEMA_ID('dbo') IS NULL EXEC('CREATE SCHEMA dbo');

CREATE TABLE actor (
    actor_id INT NOT NULL IDENTITY
    ,first_name NVARCHAR(45) NOT NULL
    ,last_name NVARCHAR(45) NOT NULL
    ,full_name AS (concat([first_name],' ',[last_name]))
    ,full_name_reversed AS (concat([last_name],' ',[first_name])) PERSISTED
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT actor_actor_id_pkey PRIMARY KEY (actor_id)
);

CREATE TABLE address (
    address_id INT NOT NULL IDENTITY
    ,address NVARCHAR(50) NOT NULL
    ,address2 NVARCHAR(50)
    ,district NVARCHAR(20) NOT NULL
    ,city_id INT NOT NULL
    ,postal_code NVARCHAR(10)
    ,phone VARCHAR(20) NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT address_address_id_pkey PRIMARY KEY (address_id)
);

CREATE TABLE category (
    category_id INT NOT NULL IDENTITY
    ,name NVARCHAR(45) NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT category_category_id_pkey PRIMARY KEY (category_id)
);

CREATE TABLE city (
    city_id INT NOT NULL IDENTITY
    ,city NVARCHAR(50) NOT NULL
    ,country_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT city_city_id_pkey PRIMARY KEY (city_id)
);

CREATE TABLE country (
    country_id INT NOT NULL IDENTITY
    ,country NVARCHAR(50) NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT country_country_id_pkey PRIMARY KEY (country_id)
);

CREATE TABLE customer (
    customer_id INT NOT NULL IDENTITY
    ,store_id INT NOT NULL
    ,first_name NVARCHAR(45) NOT NULL
    ,last_name NVARCHAR(45) NOT NULL
    ,email NVARCHAR(50)
    ,address_id INT NOT NULL
    ,active BIT NOT NULL DEFAULT 1
    ,create_date DATETIMEOFFSET NOT NULL DEFAULT (getdate())
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT customer_customer_id_pkey PRIMARY KEY (customer_id)
);

CREATE TABLE department (
    department_id BINARY(16) NOT NULL
    ,name NVARCHAR(255) NOT NULL

    ,CONSTRAINT department_department_id_pkey PRIMARY KEY (department_id)
);

CREATE TABLE employee (
    employee_id BINARY(16) NOT NULL
    ,name NVARCHAR(255) NOT NULL
    ,title NVARCHAR(255) NOT NULL
    ,manager_id BINARY(16)

    ,CONSTRAINT employee_employee_id_pkey PRIMARY KEY (employee_id)
);

CREATE TABLE employee_department (
    employee_id BINARY(16) NOT NULL
    ,department_id BINARY(16) NOT NULL

    ,CONSTRAINT employee_department_employee_id_department_id_pkey PRIMARY KEY (employee_id, department_id)
);

CREATE TABLE film (
    film_id INT NOT NULL IDENTITY
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
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT film_film_id_pkey PRIMARY KEY (film_id)
);

CREATE TABLE film_actor (
    actor_id INT NOT NULL
    ,film_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT film_actor_actor_id_film_id_pkey PRIMARY KEY (actor_id, film_id)
);

CREATE TABLE film_category (
    film_id INT NOT NULL
    ,category_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT film_category_film_id_category_id_pkey PRIMARY KEY (film_id, category_id)
);

CREATE TABLE inventory (
    inventory_id INT NOT NULL IDENTITY
    ,film_id INT NOT NULL
    ,store_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT inventory_inventory_id_pkey PRIMARY KEY (inventory_id)
);

CREATE TABLE language (
    language_id INT NOT NULL IDENTITY
    ,name CHAR(20) NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT language_language_id_pkey PRIMARY KEY (language_id)
);

CREATE TABLE payment (
    payment_id INT NOT NULL IDENTITY
    ,customer_id INT NOT NULL
    ,staff_id INT NOT NULL
    ,rental_id INT
    ,amount DECIMAL(5,2) NOT NULL
    ,payment_date DATETIMEOFFSET NOT NULL DEFAULT (getdate())
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT payment_payment_id_pkey PRIMARY KEY (payment_id)
);

CREATE TABLE rental (
    rental_id INT NOT NULL IDENTITY
    ,rental_date DATETIMEOFFSET NOT NULL DEFAULT (getdate())
    ,inventory_id INT NOT NULL
    ,customer_id INT NOT NULL
    ,return_date DATETIMEOFFSET
    ,staff_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT rental_rental_id_pkey PRIMARY KEY (rental_id)
);

CREATE TABLE staff (
    staff_id INT NOT NULL IDENTITY
    ,first_name NVARCHAR(45) NOT NULL
    ,last_name NVARCHAR(45) NOT NULL
    ,address_id INT NOT NULL
    ,picture VARBINARY(MAX)
    ,email NVARCHAR(50)
    ,store_id INT
    ,active BIT NOT NULL DEFAULT 1
    ,username NVARCHAR(16) NOT NULL
    ,password NVARCHAR(40)
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT staff_staff_id_pkey PRIMARY KEY (staff_id)
);

CREATE TABLE store (
    store_id INT NOT NULL IDENTITY
    ,manager_staff_id INT NOT NULL
    ,address_id INT NOT NULL
    ,last_update DATETIMEOFFSET NOT NULL DEFAULT (getdate())

    ,CONSTRAINT store_store_id_pkey PRIMARY KEY (store_id)
);

CREATE TABLE task (
    task_id BINARY(16) NOT NULL
    ,employee_id BINARY(16) NOT NULL
    ,department_id BINARY(16) NOT NULL
    ,task NVARCHAR(255) NOT NULL
    ,data NVARCHAR(MAX)
    ,data_deadline AS (CONVERT([char](20),json_value([data],'$.deadline')))

    ,CONSTRAINT task_task_id_pkey PRIMARY KEY (task_id)
);
