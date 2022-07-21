CREATE TABLE actor (
    actor_id INTEGER PRIMARY KEY AUTOINCREMENT
    ,first_name TEXT NOT NULL
    ,last_name TEXT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX actor_last_name_idx ON actor (last_name);

CREATE TABLE category (
    category_id INTEGER PRIMARY KEY
    ,name TEXT NOT NULL
    ,last_update TIMESTAMP NOT NULL DEFAULT (unixepoch())
);
