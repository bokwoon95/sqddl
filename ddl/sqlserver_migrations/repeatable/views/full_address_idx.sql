IF NOT EXISTS (
    SELECT 1
    FROM sys.indexes
    WHERE name = 'full_address_address_id_idx' AND object_id = OBJECT_ID('full_address')
) BEGIN
    CREATE UNIQUE CLUSTERED INDEX full_address_address_id_idx ON full_address (address_id);
END
