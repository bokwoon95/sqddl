ALTER TABLE foo.movie_award DROP CONSTRAINT IF EXISTS movie_award_best_actor_fkey;

ALTER TABLE foo.movie_award DROP CONSTRAINT IF EXISTS movie_award_best_supporting_actor_fkey;

ALTER TABLE foo.movie_award DROP CONSTRAINT IF EXISTS movie_award_best_actress_fkey;

ALTER TABLE foo.movie_award DROP CONSTRAINT IF EXISTS movie_award_best_supporting_actress_fkey;
