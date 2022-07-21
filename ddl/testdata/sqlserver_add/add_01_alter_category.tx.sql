ALTER TABLE category DROP CONSTRAINT category_category_pkey;

ALTER TABLE category ADD category_id INT NOT NULL IDENTITY;

ALTER TABLE category ALTER COLUMN category NVARCHAR(255) NULL;
