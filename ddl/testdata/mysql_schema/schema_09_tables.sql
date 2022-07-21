CREATE TABLE bar.movie (
    movie_id INT NOT NULL
    ,title VARCHAR(255)
    ,synopsis VARCHAR(255)

    ,PRIMARY KEY (movie_id)
);

CREATE INDEX movie_title_idx ON bar.movie (title);

CREATE TABLE bar.movie_actor (
    movie_id INT
    ,actor_id INT
);

CREATE TABLE bar.movie_award (
    movie_id INT
    ,best_actor INT
    ,best_supporting_actor INT
    ,best_actress INT
    ,best_supporting_actress INT
);
