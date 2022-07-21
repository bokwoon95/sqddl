DROP TABLE actor;

DROP TABLE movie;

DROP TABLE movie_award;

CREATE TABLE actors (
    actor_id INT NOT NULL IDENTITY
    ,name NVARCHAR(255)

    ,CONSTRAINT actors_actor_id_pkey PRIMARY KEY (actor_id)
);

CREATE TABLE movies (
    movie_id INT NOT NULL IDENTITY
    ,title NVARCHAR(255)
    ,synopsis NVARCHAR(255)

    ,CONSTRAINT movies_movie_id_pkey PRIMARY KEY (movie_id)
);

CREATE INDEX movies_title_idx ON movies (title);

CREATE TABLE movie_awards (
    movie_id INT
    ,best_actor INT
    ,best_supporting_actor INT
    ,best_actress INT
    ,best_supporting_actress INT
);
