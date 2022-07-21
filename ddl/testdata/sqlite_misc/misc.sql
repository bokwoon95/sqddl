PRAGMA legacy_alter_table = ON;

DROP TABLE residence;

CREATE TABLE country (
    country_id INTEGER PRIMARY KEY
    ,country TEXT
);

CREATE TABLE city (
    city_id INTEGER PRIMARY KEY
    ,city TEXT
    ,country_id INT

    ,CONSTRAINT city_country_id_fkey FOREIGN KEY (country_id) REFERENCES country (country_id) ON UPDATE CASCADE
);

CREATE INDEX city_country_id_idx ON city (country_id);

CREATE TABLE address (
    address_id INTEGER PRIMARY KEY
    ,address TEXT
    ,city_id INT

    ,CONSTRAINT address_city_id_fkey FOREIGN KEY (city_id) REFERENCES city (city_id) ON UPDATE CASCADE
);

CREATE INDEX address_city_id_idx ON address (city_id);

CREATE TABLE author_new (
    author_id INTEGER PRIMARY KEY
    ,name TEXT
    ,email TEXT NOT NULL
    ,is_active BOOLEAN

    ,CONSTRAINT author_email_key UNIQUE (email)
);
INSERT INTO author_new
    (author_id, name, email)
SELECT
    author_id, name, email
FROM
    author
;
DROP TABLE author;
ALTER TABLE author_new RENAME TO author;

CREATE INDEX author_name_idx ON author (name);

DROP INDEX post_metadata_idx;

ALTER TABLE post DROP COLUMN metadata;

ALTER TABLE post ADD COLUMN tags TEXT;

ALTER TABLE post ADD COLUMN author_id INT REFERENCES author (author_id) ON UPDATE CASCADE;

CREATE INDEX post_author_id_idx ON post (author_id);

PRAGMA legacy_alter_table = OFF;
