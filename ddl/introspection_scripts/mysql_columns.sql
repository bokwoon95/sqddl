SELECT
    c.table_schema
    ,c.table_name
    ,c.column_name
    ,c.column_type AS column_type
    ,COALESCE(c.character_maximum_length, '') AS character_length
    ,COALESCE(c.numeric_precision, '') AS numeric_precision
    ,COALESCE(c.numeric_scale, '') AS numeric_scale
    ,COALESCE(c.extra = 'auto_increment', FALSE) AS is_autoincrement
    ,c.is_nullable = 'NO' AS is_notnull
    ,COALESCE(c.extra = 'DEFAULT_GENERATED on update CURRENT_TIMESTAMP', FALSE) AS on_update_current_timestamp
    ,COALESCE(c.generation_expression, '') AS generated_expr
    ,COALESCE(c.extra = 'STORED GENERATED', FALSE) AS generated_expr_stored
    ,CASE c.collation_name
        WHEN @@collation_database THEN ''
        ELSE COALESCE(c.collation_name, '')
    END AS collation_name
    ,COALESCE(c.column_default, '') AS column_default
    ,c.column_comment
FROM
    information_schema.columns AS c
    JOIN information_schema.tables AS t USING (table_schema, table_name)
WHERE
    t.table_type = 'BASE TABLE'
    {{- if not .IncludeSystemCatalogs }}
    AND c.table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
    {{- end }}
    {{- if .Schemas }}
    AND c.table_schema IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND c.table_schema NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Tables }}
    AND c.table_name IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND c.table_name NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
ORDER BY
    c.table_schema
    ,c.table_name
    ,c.ordinal_position
;
