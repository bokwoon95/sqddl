ALTER TABLE movie
    DROP CONSTRAINT `PRIMARY`
    ,DROP CONSTRAINT movie_title_key
    ,DROP INDEX movie_category_idx
    ,DROP INDEX movie_subcategory_idx
    ,DROP COLUMN metadata
    ,MODIFY COLUMN movie_id INT
;
