ALTER TABLE movie
    ADD COLUMN metadata JSON
    ,ADD INDEX movie_category_idx (category)
    ,ADD INDEX movie_subcategory_idx (subcategory)
    ,ADD PRIMARY KEY (movie_id)
    ,ADD CONSTRAINT movie_title_key UNIQUE (title)
;
