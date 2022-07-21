ALTER TABLE movie ADD CONSTRAINT movie_subcategory_fkey FOREIGN KEY (subcategory) REFERENCES sakila.category (category);
