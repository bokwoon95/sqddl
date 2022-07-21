-- actor
ALTER TABLE actor ADD CONSTRAINT actor_actor_id_pkey PRIMARY KEY (actor_id);

-- address
ALTER TABLE address ADD CONSTRAINT address_address_id_pkey PRIMARY KEY (address_id);

-- category
ALTER TABLE category ADD CONSTRAINT category_category_id_pkey PRIMARY KEY (category_id);

-- city
ALTER TABLE city ADD CONSTRAINT city_city_id_pkey PRIMARY KEY (city_id);

-- country
ALTER TABLE country ADD CONSTRAINT country_country_id_pkey PRIMARY KEY (country_id);

-- customer
ALTER TABLE customer ADD CONSTRAINT customer_customer_id_pkey PRIMARY KEY (customer_id);
ALTER TABLE customer ADD CONSTRAINT customer_email_first_name_last_name_key UNIQUE (email, first_name, last_name);
ALTER TABLE customer ADD CONSTRAINT customer_email_key UNIQUE (email);

-- department
ALTER TABLE department ADD CONSTRAINT department_department_id_pkey PRIMARY KEY (department_id);

-- employee
ALTER TABLE employee ADD CONSTRAINT employee_employee_id_pkey PRIMARY KEY (employee_id);

-- employee_department
ALTER TABLE employee_department ADD CONSTRAINT employee_department_employee_id_department_id_pkey PRIMARY KEY (employee_id, department_id);

-- film
ALTER TABLE film ADD CONSTRAINT film_film_id_pkey PRIMARY KEY (film_id);

-- film_actor
ALTER TABLE film_actor ADD CONSTRAINT film_actor_actor_id_film_id_pkey PRIMARY KEY (actor_id, film_id);

-- film_category
ALTER TABLE film_category ADD CONSTRAINT film_category_film_id_category_id_pkey PRIMARY KEY (film_id, category_id);

-- inventory
ALTER TABLE inventory ADD CONSTRAINT inventory_inventory_id_pkey PRIMARY KEY (inventory_id);

-- language
ALTER TABLE language ADD CONSTRAINT language_language_id_pkey PRIMARY KEY (language_id);

-- payment
ALTER TABLE payment ADD CONSTRAINT payment_payment_id_pkey PRIMARY KEY (payment_id);

-- rental
ALTER TABLE rental ADD CONSTRAINT rental_no_overlap EXCLUDE USING gist (inventory_id WITH =, tstzrange(rental_date, return_date) WITH &&);
ALTER TABLE rental ADD CONSTRAINT rental_rental_id_pkey PRIMARY KEY (rental_id);

-- staff
ALTER TABLE staff ADD CONSTRAINT staff_email_key UNIQUE (email);
ALTER TABLE staff ADD CONSTRAINT staff_staff_id_pkey PRIMARY KEY (staff_id);

-- store
ALTER TABLE store ADD CONSTRAINT store_store_id_pkey PRIMARY KEY (store_id);

-- task
ALTER TABLE task ADD CONSTRAINT task_task_id_pkey PRIMARY KEY (task_id);

-- actor

-- address
ALTER TABLE address ADD CONSTRAINT address_city_id_fkey FOREIGN KEY (city_id) REFERENCES city (city_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

-- category

-- city
ALTER TABLE city ADD CONSTRAINT city_country_id_fkey FOREIGN KEY (country_id) REFERENCES country (country_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

-- country

-- customer
ALTER TABLE customer ADD CONSTRAINT customer_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
ALTER TABLE customer ADD CONSTRAINT customer_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

-- department

-- employee
ALTER TABLE employee ADD CONSTRAINT employee_manager_id_fkey FOREIGN KEY (manager_id) REFERENCES employee (employee_id);

-- employee_department
ALTER TABLE employee_department ADD CONSTRAINT employee_department_department_id_fkey FOREIGN KEY (department_id) REFERENCES department (department_id);
ALTER TABLE employee_department ADD CONSTRAINT employee_department_employee_id_fkey FOREIGN KEY (employee_id) REFERENCES employee (employee_id);

-- film
ALTER TABLE film ADD CONSTRAINT film_language_id_fkey FOREIGN KEY (language_id) REFERENCES language (language_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
ALTER TABLE film ADD CONSTRAINT film_original_language_id_fkey FOREIGN KEY (original_language_id) REFERENCES language (language_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

-- film_actor
ALTER TABLE film_actor ADD CONSTRAINT film_actor_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES actor (actor_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
ALTER TABLE film_actor ADD CONSTRAINT film_actor_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

-- film_category
ALTER TABLE film_category ADD CONSTRAINT film_category_category_id_fkey FOREIGN KEY (category_id) REFERENCES category (category_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
ALTER TABLE film_category ADD CONSTRAINT film_category_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

-- inventory
ALTER TABLE inventory ADD CONSTRAINT inventory_film_id_fkey FOREIGN KEY (film_id) REFERENCES film (film_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
ALTER TABLE inventory ADD CONSTRAINT inventory_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

-- language

-- payment
ALTER TABLE payment ADD CONSTRAINT payment_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer (customer_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
ALTER TABLE payment ADD CONSTRAINT payment_rental_id_fkey FOREIGN KEY (rental_id) REFERENCES rental (rental_id) ON UPDATE CASCADE ON DELETE SET NULL DEFERRABLE;
ALTER TABLE payment ADD CONSTRAINT payment_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff (staff_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

-- rental
ALTER TABLE rental ADD CONSTRAINT rental_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES customer (customer_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
ALTER TABLE rental ADD CONSTRAINT rental_inventory_id_fkey FOREIGN KEY (inventory_id) REFERENCES inventory (inventory_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
ALTER TABLE rental ADD CONSTRAINT rental_staff_id_fkey FOREIGN KEY (staff_id) REFERENCES staff (staff_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

-- staff
ALTER TABLE staff ADD CONSTRAINT staff_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
ALTER TABLE staff ADD CONSTRAINT staff_store_id_fkey FOREIGN KEY (store_id) REFERENCES store (store_id) DEFERRABLE;

-- store
ALTER TABLE store ADD CONSTRAINT store_address_id_fkey FOREIGN KEY (address_id) REFERENCES address (address_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;
ALTER TABLE store ADD CONSTRAINT store_manager_staff_id_fkey FOREIGN KEY (manager_staff_id) REFERENCES staff (staff_id) ON UPDATE CASCADE ON DELETE RESTRICT DEFERRABLE;

-- task
ALTER TABLE task ADD CONSTRAINT task_employee_id_department_id_fkey FOREIGN KEY (employee_id, department_id) REFERENCES employee_department (employee_id, department_id);
