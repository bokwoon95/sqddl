ALTER TABLE category
    DROP CONSTRAINT `PRIMARY`
    ,ADD COLUMN category_id INT NOT NULL
    ,MODIFY COLUMN category VARCHAR(255)
    ,ADD PRIMARY KEY (category_id)
    ,ADD CONSTRAINT category_category_key UNIQUE (category)
;
