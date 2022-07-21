CREATE TABLE actor (
    actor_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,first_name TEXT NOT NULL
    ,last_name TEXT NOT NULL
    ,full_name TEXT GENERATED ALWAYS AS (first_name || ' ' || last_name) VIRTUAL
    ,full_name_reversed TEXT GENERATED ALWAYS AS (last_name || ' ' || first_name) VIRTUAL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE address (
    address_id INTEGER PRIMARY KEY
    ,address TEXT NOT NULL
    ,address2 TEXT
    ,district TEXT NOT NULL
    ,city_id INT NOT NULL
    ,postal_code TEXT
    ,phone TEXT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT address_city_id_fkey FOREIGN KEY (city_id) REFERENCES city (city_id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE category (
    category_id INTEGER PRIMARY KEY
    ,name TEXT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE city (
    city_id INTEGER PRIMARY KEY
    ,city TEXT NOT NULL
    ,country_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT city_country_id_fkey FOREIGN KEY (country_id) REFERENCES country (country_id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE country (
    country_id INTEGER PRIMARY KEY
    ,country TEXT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE customer (
    customer_id INTEGER PRIMARY KEY
    ,store_id INT NOT NULL
    ,first_name TEXT NOT NULL
    ,last_name TEXT NOT NULL
    ,email TEXT
    ,address_id INT NOT NULL
    ,active INT NOT NULL DEFAULT TRUE
    ,create_date TIMESTAMP NOT NULL DEFAULT (unixepoch())
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT customer_email_first_name_last_name_key UNIQUE (email, first_name, last_name)
    ,CONSTRAINT customer_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT customer_email_key UNIQUE (email)
    ,CONSTRAINT customer_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE department (
    department_id UUID NOT NULL
    ,name TEXT NOT NULL

    ,CONSTRAINT department_department_id_pkey PRIMARY KEY (department_id)
);

CREATE TABLE employee (
    employee_id UUID NOT NULL
    ,name TEXT NOT NULL
    ,title TEXT NOT NULL
    ,manager_id UUID

    ,CONSTRAINT employee_employee_id_pkey PRIMARY KEY (employee_id)
    ,CONSTRAINT employee_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES employee (employee_id)
);

CREATE TABLE employee_department (
    employee_id UUID NOT NULL
    ,department_id UUID NOT NULL

    ,CONSTRAINT employee_department_employee_id_department_id_pkey PRIMARY KEY (employee_id, department_id)
    ,CONSTRAINT employee_department_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES employee (employee_id)
    ,CONSTRAINT employee_department_department_id_fkey FOREIGN KEY (department_id) REFERENCES department (department_id)
);

CREATE TABLE film (
    film_id INTEGER PRIMARY KEY
    ,title TEXT NOT NULL
    ,description TEXT
    ,release_year INT
    ,language_id INT NOT NULL
    ,original_language_id INT
    ,rental_duration INT NOT NULL DEFAULT 3
    ,rental_rate REAL NOT NULL DEFAULT 4.99
    ,length INT
    ,replacement_cost REAL NOT NULL DEFAULT 19.99
    ,rating TEXT DEFAULT 'G'
    ,special_features JSON
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT film_year_check CHECK (release_year >= 1901 AND release_year <= 2155)
    ,CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT film_original_language_id_fkey FOREIGN KEY (original_language_id) REFERENCES language (language_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT film_rating_check CHECK (rating IN ('G','PG','PG-13','R','NC-17'))
);

CREATE TABLE film_actor (
    actor_id INT NOT NULL
    ,film_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT film_actor_actor_id_film_id_pkey PRIMARY KEY (actor_id, film_id)
    ,CONSTRAINT film_actor_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES actor (actor_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT film_actor_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE film_category (
    film_id INT NOT NULL
    ,category_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT film_category_film_id_category_id_pkey PRIMARY KEY (film_id, category_id)
    ,CONSTRAINT film_category_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT film_category_category_id_fkey FOREIGN KEY (category_id) REFERENCES category (category_id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE VIRTUAL TABLE film_text USING FTS5 (
    title
    ,description
    ,content='film'
    ,content_rowid='film_id'
);

CREATE TABLE inventory (
    inventory_id INTEGER PRIMARY KEY
    ,film_id INT NOT NULL
    ,store_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT inventory_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT inventory_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE language (
    language_id INTEGER PRIMARY KEY
    ,name TEXT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE payment (
    payment_id INTEGER PRIMARY KEY
    ,customer_id INT NOT NULL
    ,staff_id INT NOT NULL
    ,rental_id INT
    ,amount REAL NOT NULL
    ,payment_date TIMESTAMP NOT NULL DEFAULT (unixepoch())
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT payment_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer (customer_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT payment_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff (staff_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT payment_rental_id_fkey FOREIGN KEY (rental_id) REFERENCES rental (rental_id) ON UPDATE CASCADE ON DELETE SET NULL
);

CREATE TABLE rental (
    rental_id INTEGER PRIMARY KEY
    ,rental_date TIMESTAMP NOT NULL DEFAULT (unixepoch())
    ,inventory_id INT NOT NULL
    ,customer_id INT NOT NULL
    ,return_date TIMESTAMP
    ,staff_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT rental_inventory_id_fkey FOREIGN KEY (inventory_id) REFERENCES inventory (inventory_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT rental_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer (customer_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT rental_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff (staff_id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE staff (
    staff_id INTEGER PRIMARY KEY
    ,first_name TEXT NOT NULL
    ,last_name TEXT NOT NULL
    ,address_id INT NOT NULL
    ,picture BLOB
    ,email TEXT
    ,store_id INT
    ,active INT NOT NULL DEFAULT TRUE
    ,username TEXT NOT NULL
    ,password TEXT
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT staff_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT staff_email_key UNIQUE (email)
    ,CONSTRAINT staff_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE store (
    store_id INTEGER PRIMARY KEY
    ,manager_staff_id INT NOT NULL
    ,address_id INT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())

    ,CONSTRAINT store_manager_staff_id_fkey FOREIGN KEY (manager_staff_id) REFERENCES staff (staff_id) ON UPDATE CASCADE ON DELETE RESTRICT
    ,CONSTRAINT store_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE TABLE task (
    task_id UUID NOT NULL
    ,employee_id UUID NOT NULL
    ,department_id UUID NOT NULL
    ,task TEXT NOT NULL
    ,data JSON

    ,CONSTRAINT task_task_id_pkey PRIMARY KEY (task_id)
    ,CONSTRAINT task_employee_id_department_id_fkey FOREIGN KEY (employee_id, department_id) REFERENCES employee_department (employee_id, department_id)
);
