DROP INDEX movie_category_idx ON movie;

DROP INDEX movie_subcategory_idx ON movie;

ALTER TABLE movie DROP CONSTRAINT movie_movie_id_pkey;

ALTER TABLE movie DROP CONSTRAINT movie_title_key;

ALTER TABLE movie DROP COLUMN metadata;

ALTER TABLE movie ALTER COLUMN movie_id INT NOT NULL;
