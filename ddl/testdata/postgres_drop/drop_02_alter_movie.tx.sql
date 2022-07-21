DROP INDEX IF EXISTS movie_category_idx;

DROP INDEX IF EXISTS movie_subcategory_idx;

ALTER TABLE movie DROP CONSTRAINT IF EXISTS movie_movie_id_pkey;

ALTER TABLE movie DROP CONSTRAINT IF EXISTS movie_title_key;

ALTER TABLE movie DROP COLUMN IF EXISTS metadata;

ALTER TABLE movie ALTER COLUMN movie_id DROP NOT NULL;
