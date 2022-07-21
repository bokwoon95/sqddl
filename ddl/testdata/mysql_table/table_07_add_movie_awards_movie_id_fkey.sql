ALTER TABLE movie_awards ADD CONSTRAINT movie_awards_movie_id_fkey FOREIGN KEY (movie_id) REFERENCES sakila.movies (movie_id);
