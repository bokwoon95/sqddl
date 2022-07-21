-- actor
CREATE INDEX actor_last_name_idx ON actor (last_name);

-- address
CREATE INDEX address_city_id_idx ON address (city_id);

-- category

-- city
CREATE INDEX city_country_id_idx ON city (country_id);

-- country

-- customer
CREATE INDEX customer_address_id_idx ON customer (address_id);
CREATE INDEX customer_last_name_idx ON customer (last_name);
CREATE INDEX customer_store_id_idx ON customer (store_id);

-- department

-- employee
CREATE INDEX employee_manager_id_idx ON employee (manager_id);

-- employee_department
CREATE INDEX employee_department_department_id_idx ON employee_department (department_id);
CREATE INDEX employee_department_employee_id_idx ON employee_department (employee_id);

-- film
CREATE INDEX film_language_id_idx ON film (language_id);
CREATE INDEX film_original_language_id_idx ON film (original_language_id);
CREATE INDEX film_title_idx ON film (title);

-- film_actor
CREATE INDEX film_actor_film_id_idx ON film_actor (film_id);

-- film_category

-- film_text
CREATE FULLTEXT INDEX film_text_title_description_idx ON film_text (title, description);

-- inventory
CREATE INDEX inventory_film_id_idx ON inventory (film_id);
CREATE INDEX inventory_store_id_film_id_idx ON inventory (store_id, film_id);

-- language

-- payment
CREATE INDEX payment_customer_id_idx ON payment (customer_id);
CREATE INDEX payment_rental_id_idx ON payment (rental_id);
CREATE INDEX payment_staff_id_idx ON payment (staff_id);

-- rental
CREATE INDEX rental_customer_id_idx ON rental (customer_id);
CREATE INDEX rental_inventory_id_idx ON rental (inventory_id);
CREATE INDEX rental_staff_id_idx ON rental (staff_id);

-- staff
CREATE INDEX staff_address_id_idx ON staff (address_id);
CREATE INDEX staff_store_id_idx ON staff (store_id);

-- store
CREATE INDEX store_address_id_idx ON store (address_id);
CREATE INDEX store_manager_staff_id_idx ON store (manager_staff_id);

-- task
CREATE INDEX task_data_idx ON task ((cast(json_unquote(json_extract(`data`,_utf8mb4'$.deadline')) as char(20) charset utf8mb4)) DESC);
CREATE INDEX task_employee_id_department_id_idx ON task (employee_id, department_id);
CREATE INDEX task_task_idx ON task (task DESC);
