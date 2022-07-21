ALTER TABLE bar.movie_award ADD CONSTRAINT movie_award_best_actor_fkey FOREIGN KEY (best_actor) REFERENCES actor (actor_id);

ALTER TABLE bar.movie_award ADD CONSTRAINT movie_award_best_supporting_actor_fkey FOREIGN KEY (best_supporting_actor) REFERENCES actor (actor_id);

ALTER TABLE bar.movie_award ADD CONSTRAINT movie_award_best_actress_fkey FOREIGN KEY (best_actress) REFERENCES actor (actor_id);

ALTER TABLE bar.movie_award ADD CONSTRAINT movie_award_best_supporting_actress_fkey FOREIGN KEY (best_supporting_actress) REFERENCES actor (actor_id);
