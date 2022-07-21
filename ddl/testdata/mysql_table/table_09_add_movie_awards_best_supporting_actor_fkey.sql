ALTER TABLE movie_awards ADD CONSTRAINT movie_awards_best_supporting_actor_fkey FOREIGN KEY (best_supporting_actor) REFERENCES sakila.actors (actor_id);
