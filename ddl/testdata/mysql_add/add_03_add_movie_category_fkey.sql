ALTER TABLE movie ADD CONSTRAINT movie_category_fkey FOREIGN KEY (category) REFERENCES sakila.category (category);
