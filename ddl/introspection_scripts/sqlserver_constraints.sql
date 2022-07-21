SELECT
    '' AS table_schema
    ,'' AS table_name
    ,'' AS constraint_name
    ,'' AS constraint_type
    ,'' AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS check_expr
    ,0 AS is_clustered
    ,0 AS is_not_valid
WHERE
    1 <> 1
{{- if .IncludeConstraintType "PRIMARY KEY" }}
UNION ALL
SELECT
    schemas.name AS table_schema
    ,tables.name AS table_name
    ,indexes.name AS constraint_name
    ,'PRIMARY KEY' AS constraint_type
    ,string_agg(columns.name, ',') WITHIN GROUP (ORDER BY index_columns.key_ordinal) AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS check_expr
    ,CASE WHEN indexes.index_id = 1 THEN 1 ELSE 0 END AS is_clustered
    ,0 AS is_not_valid
FROM
    sys.indexes
    JOIN sys.tables ON tables.object_id = indexes.object_id
    JOIN sys.schemas ON schemas.schema_id = tables.schema_id
    JOIN sys.index_columns
        ON index_columns.index_id = indexes.index_id
        AND index_columns.object_id = indexes.object_id
    JOIN sys.columns
        ON columns.column_id = index_columns.column_id
        AND columns.object_id = indexes.object_id
WHERE
    indexes.is_primary_key = 1
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
GROUP BY
    schemas.name
    ,tables.name
    ,indexes.name
    ,indexes.index_id
{{- end }}
{{- if .IncludeConstraintType "UNIQUE" }}
UNION ALL
SELECT
    schemas.name AS table_schema
    ,tables.name AS table_name
    ,indexes.name AS constraint_name
    ,'UNIQUE' AS constraint_type
    ,string_agg(columns.name, ',') WITHIN GROUP (ORDER BY index_columns.key_ordinal) AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS check_expr
    ,CASE WHEN indexes.index_id = 1 THEN 1 ELSE 0 END AS is_clustered
    ,0 AS is_not_valid
FROM
    sys.indexes
    JOIN sys.tables ON tables.object_id = indexes.object_id
    JOIN sys.schemas ON schemas.schema_id = tables.schema_id
    JOIN sys.index_columns
        ON index_columns.index_id = indexes.index_id
        AND index_columns.object_id = indexes.object_id
    JOIN sys.columns
        ON columns.column_id = index_columns.column_id
        AND columns.object_id = indexes.object_id
WHERE
    indexes.is_unique_constraint = 1
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
GROUP BY
    schemas.name
    ,tables.name
    ,indexes.name
    ,indexes.index_id
{{- end }}
{{- if .IncludeConstraintType "FOREIGN KEY" }}
UNION ALL
SELECT
    table_schema
    ,table_name
    ,constraint_name
    ,'FOREIGN KEY' AS constraint_type
    ,string_agg(column_name, ',') WITHIN GROUP (ORDER BY seq) AS columns
    ,references_schema
    ,references_table
    ,string_agg(references_column, ',') WITHIN GROUP (ORDER BY seq) AS references_columns
    ,update_rule
    ,delete_rule
    ,'' AS check_expr
    ,0 AS is_clustered
    ,is_not_valid
FROM (
    SELECT
        schemas1.name AS table_schema
        ,tables1.name AS table_name
        ,foreign_keys.name AS constraint_name
        ,columns1.name AS column_name
        ,schemas2.name AS references_schema
        ,tables2.name AS references_table
        ,columns2.name AS references_column
        ,foreign_keys.update_referential_action_desc AS update_rule
        ,foreign_keys.delete_referential_action_desc AS delete_rule
        ,foreign_keys.is_not_trusted AS is_not_valid
        ,foreign_key_columns.constraint_column_id AS seq
    FROM
        sys.foreign_keys
        JOIN sys.tables AS tables1 ON tables1.object_id = foreign_keys.parent_object_id
        JOIN sys.schemas AS schemas1 ON schemas1.schema_id = tables1.schema_id
        JOIN sys.foreign_key_columns ON foreign_key_columns.constraint_object_id = foreign_keys.object_id
        JOIN sys.columns AS columns1
            ON columns1.object_id = foreign_key_columns.parent_object_id
            AND columns1.column_id = foreign_key_columns.parent_column_id
        JOIN sys.columns AS columns2
            ON columns2.object_id = foreign_key_columns.referenced_object_id
            AND columns2.column_id = foreign_key_columns.referenced_column_id
        JOIN sys.tables AS tables2 ON tables2.object_id = foreign_key_columns.referenced_object_id
        JOIN sys.schemas AS schemas2 ON schemas2.schema_id = tables2.schema_id
    WHERE
        1 = 1
        {{- if not .IncludeSystemCatalogs }}
        AND tables1.name NOT LIKE 'spt_%' AND tables1.name NOT LIKE 'msreplication_%'
        {{- end }}
        {{- if .Schemas }}
        AND schemas1.name IN ({{ mklist .Schemas }})
        {{- else if .ExcludeSchemas }}
        AND schemas1.name NOT IN ({{ mklist .ExcludeSchemas }})
        {{- end }}
        {{- if .Tables }}
        AND tables1.name IN ({{ mklist .Tables }})
        {{- else if .ExcludeTables }}
        AND tables1.name NOT IN ({{ mklist .ExcludeTables }})
        {{- end }}
    ) AS foreign_key_columns
GROUP BY
    table_schema
    ,table_name
    ,constraint_name
    ,references_schema
    ,references_table
    ,update_rule
    ,delete_rule
    ,is_not_valid
{{- end }}
{{- if .IncludeConstraintType "CHECK" }}
UNION ALL
SELECT
    schemas.name AS table_schema
    ,tables.name AS table_name
    ,check_constraints.name AS constraint_name
    ,'CHECK' AS constraint_type
    ,'' AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,check_constraints.definition AS check_expr
    ,0 AS is_clustered
    ,check_constraints.is_not_trusted AS is_not_valid
FROM
    sys.check_constraints
    JOIN sys.tables ON tables.object_id = check_constraints.parent_object_id
    JOIN sys.schemas ON schemas.schema_id = tables.schema_id
WHERE
    1 = 1
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
{{- end }}
ORDER BY
    table_schema
    ,table_name
    ,constraint_name
;
