ALTER TABLE bar.movie_award ADD CONSTRAINT movie_award_best_actress_fkey FOREIGN KEY (best_actress) REFERENCES sakila.actor (actor_id);
