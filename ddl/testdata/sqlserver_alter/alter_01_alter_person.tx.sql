ALTER TABLE person DROP CONSTRAINT person_person_id_pkey;

ALTER TABLE person ALTER COLUMN person_id NVARCHAR(255) NOT NULL;

ALTER TABLE person ALTER COLUMN name TEXT;

ALTER TABLE person ALTER COLUMN email VARCHAR(255) COLLATE Latin1_General_100_BIN2_UTF8;

BEGIN
    DECLARE @name NVARCHAR(255);
    SELECT
        @name = default_constraints.name
    FROM
        sys.default_constraints
        JOIN sys.tables ON tables.object_id = default_constraints.parent_object_id
        JOIN sys.columns ON columns.column_id = default_constraints.parent_column_id
    WHERE
        SCHEMA_NAME(tables.schema_id) = 'dbo'
        AND tables.name = 'person'
        AND columns.name = 'password'
    ;
    EXEC('ALTER TABLE person DROP CONSTRAINT ' + @name);
END;

ALTER TABLE person ALTER COLUMN bio VARCHAR(255);

ALTER TABLE person ADD DEFAULT 'lorem ipsum' FOR bio;

ALTER TABLE person ALTER COLUMN notes VARCHAR(1000);

ALTER TABLE person ALTER COLUMN height_meters NUMERIC(3,2);

ALTER TABLE person ALTER COLUMN weight_kilos NUMERIC(3,2);

ALTER TABLE person ALTER COLUMN salary_dollars DECIMAL(10,2);

ALTER TABLE person ALTER COLUMN ip_address VARCHAR(45);
