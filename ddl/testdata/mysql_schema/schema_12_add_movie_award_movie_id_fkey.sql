ALTER TABLE bar.movie_award ADD CONSTRAINT movie_award_movie_id_fkey FOREIGN KEY (movie_id) REFERENCES bar.movie (movie_id);
