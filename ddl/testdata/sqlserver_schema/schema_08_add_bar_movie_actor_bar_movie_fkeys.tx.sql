ALTER TABLE bar.movie_actor ADD CONSTRAINT movie_actor_movie_id_fkey FOREIGN KEY (movie_id) REFERENCES bar.movie (movie_id);
