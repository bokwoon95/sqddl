ALTER TABLE bar.movie_actor ADD CONSTRAINT movie_actor_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES actor (actor_id);
