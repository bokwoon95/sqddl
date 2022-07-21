SELECT
    schemas.name AS table_schema
    ,tables.name AS table_name
    ,columns.name AS column_name
    ,COALESCE(TYPE_NAME(columns.system_type_id), tables.name) AS column_type
    ,COALESCE(COLUMNPROPERTY(columns.object_id, columns.name, 'charmaxlen'), '') AS character_length
    ,CASE
        WHEN columns.system_type_id IN (48, 52, 56, 59, 60, 62, 106, 108, 122, 127) THEN columns.precision
        ELSE ''
    END AS numeric_precision
    ,CASE
        WHEN columns.system_type_id IN (40, 41, 42, 43, 58, 61) THEN ''
        ELSE COALESCE(ODBCSCALE(columns.system_type_id, columns.scale), '')
    END AS numeric_scale
    ,CASE
        WHEN columns.is_identity = 1
            THEN CASE
                WHEN identity_columns.seed_value = 1 AND identity_columns.increment_value = 1 THEN 'IDENTITY'
                ELSE CONCAT('IDENTITY(', CAST(identity_columns.seed_value AS VARCHAR(255)), ', ', CAST(identity_columns.increment_value AS VARCHAR(255)), ')')
            END
        ELSE ''
    END AS column_identity
    ,CASE WHEN columns.is_nullable = 1 THEN 0 ELSE 1 END AS is_notnull
    ,COALESCE(computed_columns.definition, '') AS generated_expr
    ,COALESCE(computed_columns.is_persisted, 0) AS generated_expr_stored
    ,CASE columns.collation_name
        WHEN SERVERPROPERTY('collation') THEN ''
        ELSE COALESCE(columns.collation_name, '')
    END AS collation_name
    ,COALESCE(OBJECT_DEFINITION(columns.default_object_id), '') AS column_default
FROM
    sys.columns
    JOIN sys.tables ON tables.object_id = columns.object_id
    JOIN sys.schemas ON schemas.schema_id = tables.schema_id
    LEFT JOIN sys.types ON types.user_type_id = columns.user_type_id
    LEFT JOIN sys.identity_columns
        ON identity_columns.object_id = columns.object_id
        AND identity_columns.column_id = columns.column_id
    LEFT JOIN sys.computed_columns
        ON computed_columns.object_id = columns.object_id
        AND computed_columns.column_id = columns.column_id
WHERE
    tables.type = 'U' -- User-defined table (https://stackoverflow.com/a/2907204)
    {{- if not .IncludeSystemCatalogs }}
    AND tables.name NOT LIKE 'spt_%' AND tables.name NOT LIKE 'msreplication_%'
    {{- end }}
    {{- if .Schemas }}
    AND schemas.name IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND schemas.name NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Tables }}
    AND tables.name IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND tables.name NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
ORDER BY
    schemas.name
    ,tables.name
    ,COLUMNPROPERTY(columns.object_id, columns.name, 'ordinal')
;
