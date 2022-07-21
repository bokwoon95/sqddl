ALTER TABLE movie ADD CONSTRAINT movie_category_fkey FOREIGN KEY (category) REFERENCES category (category) NOT VALID;

ALTER TABLE movie ADD CONSTRAINT movie_subcategory_fkey FOREIGN KEY (subcategory) REFERENCES category (category) NOT VALID;
