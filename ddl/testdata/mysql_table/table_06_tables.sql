DROP TABLE IF EXISTS actor;

DROP TABLE IF EXISTS movie;

DROP TABLE IF EXISTS movie_award;

CREATE TABLE actors (
    actor_id INT NOT NULL
    ,name VARCHAR(255)

    ,PRIMARY KEY (actor_id)
);

CREATE TABLE movies (
    movie_id INT NOT NULL
    ,title VARCHAR(255)
    ,synopsis VARCHAR(255)

    ,PRIMARY KEY (movie_id)
);

CREATE INDEX movies_title_idx ON movies (title);

CREATE TABLE movie_awards (
    movie_id INT
    ,best_actor INT
    ,best_supporting_actor INT
    ,best_actress INT
    ,best_supporting_actress INT
);
