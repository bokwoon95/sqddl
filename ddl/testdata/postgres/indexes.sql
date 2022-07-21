-- actor
CREATE INDEX actor_last_name_idx ON actor USING btree (last_name);

-- address
CREATE INDEX address_city_id_idx ON address USING btree (city_id);

-- category

-- city
CREATE INDEX city_country_id_idx ON city USING btree (country_id);

-- country

-- customer
CREATE INDEX customer_address_id_idx ON customer USING btree (address_id);
CREATE INDEX customer_last_name_idx ON customer USING btree (last_name);
CREATE INDEX customer_store_id_idx ON customer USING btree (store_id);

-- department

-- employee
CREATE INDEX employee_manager_id_idx ON employee USING btree (manager_id);

-- employee_department
CREATE INDEX employee_department_department_id_idx ON employee_department USING btree (department_id);
CREATE INDEX employee_department_employee_id_idx ON employee_department USING btree (employee_id);

-- film
CREATE INDEX film_fulltext_idx ON film USING gin (fulltext);
CREATE INDEX film_language_id_idx ON film USING btree (language_id);
CREATE INDEX film_original_language_id_idx ON film USING btree (original_language_id);
CREATE INDEX film_title_idx ON film USING btree (title);

-- film_actor
CREATE INDEX film_actor_film_id_idx ON film_actor USING btree (film_id);

-- film_category

-- inventory
CREATE INDEX inventory_film_id_idx ON inventory USING btree (film_id);
CREATE INDEX inventory_store_id_film_id_idx ON inventory USING btree (store_id, film_id);

-- language

-- payment
CREATE INDEX payment_customer_id_idx ON payment USING btree (customer_id);
CREATE INDEX payment_rental_id_idx ON payment USING btree (rental_id);
CREATE INDEX payment_staff_id_idx ON payment USING btree (staff_id);

-- rental
CREATE INDEX rental_customer_id_idx ON rental USING btree (customer_id);
CREATE UNIQUE INDEX rental_inventory_id_customer_id_staff_id_idx ON rental USING btree (inventory_id, customer_id, staff_id);
CREATE INDEX rental_inventory_id_idx ON rental USING btree (inventory_id);
CREATE INDEX rental_staff_id_idx ON rental USING btree (staff_id);

-- staff
CREATE INDEX staff_address_id_idx ON staff USING btree (address_id);
CREATE INDEX staff_store_id_idx ON staff USING btree (store_id);

-- store
CREATE INDEX store_address_id_idx ON store USING btree (address_id);
CREATE INDEX store_manager_staff_id_idx ON store USING btree (manager_staff_id);

-- task
CREATE INDEX task_data_idx ON task USING btree ((data ->> 'deadline'::text) DESC) WHERE data IS NOT NULL;
CREATE INDEX task_employee_id_department_id_idx ON task USING btree (employee_id, department_id);
CREATE INDEX task_task_idx ON task USING btree (task COLLATE "C" varchar_pattern_ops DESC) INCLUDE (employee_id, department_id);
