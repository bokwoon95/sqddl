ALTER TABLE movie_awards ADD CONSTRAINT movie_awards_best_actor_fkey FOREIGN KEY (best_actor) REFERENCES actors (actor_id);

ALTER TABLE movie_awards ADD CONSTRAINT movie_awards_best_supporting_actor_fkey FOREIGN KEY (best_supporting_actor) REFERENCES actors (actor_id);

ALTER TABLE movie_awards ADD CONSTRAINT movie_awards_best_actress_fkey FOREIGN KEY (best_actress) REFERENCES actors (actor_id);

ALTER TABLE movie_awards ADD CONSTRAINT movie_awards_best_supporting_actress_fkey FOREIGN KEY (best_supporting_actress) REFERENCES actors (actor_id);
