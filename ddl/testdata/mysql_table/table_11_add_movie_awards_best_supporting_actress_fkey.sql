ALTER TABLE movie_awards ADD CONSTRAINT movie_awards_best_supporting_actress_fkey FOREIGN KEY (best_supporting_actress) REFERENCES sakila.actors (actor_id);
