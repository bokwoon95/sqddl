ALTER TABLE bar.movie_award ADD CONSTRAINT movie_award_best_supporting_actor_fkey FOREIGN KEY (best_supporting_actor) REFERENCES sakila.actor (actor_id);
