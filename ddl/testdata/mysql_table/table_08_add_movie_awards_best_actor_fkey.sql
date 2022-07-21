ALTER TABLE movie_awards ADD CONSTRAINT movie_awards_best_actor_fkey FOREIGN KEY (best_actor) REFERENCES sakila.actors (actor_id);
