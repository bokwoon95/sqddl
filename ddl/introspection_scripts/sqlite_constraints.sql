SELECT
    '' AS table_name
    ,'' AS constraint_type
    ,'' AS columns
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
WHERE
    FALSE
{{- if .IncludeConstraintType "PRIMARY KEY" }}
UNION ALL
SELECT
    table_name
    ,'PRIMARY KEY' AS constraint_type
    ,COALESCE(group_concat(column_name), 'ROWID') AS columns
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
FROM (
    SELECT
        tables.tbl_name AS table_name
        ,columns.name AS column_name
    FROM (
        SELECT
            tbl_name
        FROM
            sqlite_schema
        WHERE
            type = 'table'
            {{- if not .IncludeSystemCatalogs }}
            AND tbl_name NOT LIKE 'sqlite_%' AND sql NOT LIKE 'CREATE TABLE ''%'
            {{- end }}
            {{- if .Tables }}
            AND tbl_name IN ({{ mklist .Tables }})
            {{- else if .ExcludeTables }}
            AND tbl_name NOT IN ({{ mklist .ExcludeTables }})
            {{- end }}
        ) AS tables
        CROSS JOIN pragma_table_info(tables.tbl_name) AS columns
    WHERE
        columns.pk > 0 -- exclude non-primarykey columns
    ORDER BY
        tables.tbl_name
        ,columns.pk
    ) AS primary_key_columns
GROUP BY
    table_name
{{- end }}
{{- if .IncludeConstraintType "UNIQUE" }}
UNION ALL
SELECT
    table_name
    ,'UNIQUE' AS constraint_type
    ,COALESCE(group_concat(column_name), '') AS columns
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
FROM (
    SELECT
        tables.tbl_name AS table_name
        ,indexes.name AS index_name
        ,columns.name AS column_name
    FROM (
        SELECT
            tbl_name
        FROM
            sqlite_schema
        WHERE
            type = 'table'
            {{- if not .IncludeSystemCatalogs }}
            AND tbl_name NOT LIKE 'sqlite_%' AND sql NOT LIKE 'CREATE TABLE ''%'
            {{- end }}
            {{- if .Tables }}
            AND tbl_name IN ({{ mklist .Tables }})
            {{- else if .ExcludeTables }}
            AND tbl_name NOT IN ({{ mklist .ExcludeTables }})
            {{- end }}
        ) AS tables
        CROSS JOIN pragma_index_list(tables.tbl_name) AS indexes
        CROSS JOIN pragma_index_info(indexes.name) AS columns
    WHERE
        indexes."unique"
        AND indexes.origin = 'u'
    ORDER BY
        columns.seqno
    ) AS unique_columns
GROUP BY
    table_name
    ,index_name
{{- end }}
{{- if .IncludeConstraintType "FOREIGN KEY" }}
UNION ALL
SELECT
    table_name
    ,'FOREIGN KEY' AS constraint_type
    ,COALESCE(group_concat(column_name), '') AS columns
    ,references_table
    ,COALESCE(group_concat(references_column), '') AS references_columns
    ,update_rule
    ,delete_rule
FROM (
    SELECT
        tables.tbl_name AS table_name
        ,columns.id AS foreign_key_id
        ,columns."from" AS column_name
        ,columns."table" AS references_table
        ,columns."to" AS references_column
        ,columns.on_update AS update_rule
        ,columns.on_delete AS delete_rule
    FROM (
        SELECT
            tbl_name
        FROM
            sqlite_schema
        WHERE
            type = 'table'
            {{- if not .IncludeSystemCatalogs }}
            AND tbl_name NOT LIKE 'sqlite_%' AND sql NOT LIKE 'CREATE TABLE ''%'
            {{- end }}
            {{- if .Tables }}
            AND tbl_name IN ({{ mklist .Tables }})
            {{- else if .ExcludeTables }}
            AND tbl_name NOT IN ({{ mklist .ExcludeTables }})
            {{- end }}
        ) AS tables
        CROSS JOIN pragma_foreign_key_list(tables.tbl_name) AS columns
    ORDER BY
        columns.seq
    ) AS foreign_key_columns
GROUP BY
    table_name
    ,foreign_key_id
    ,references_table
    ,update_rule
    ,delete_rule
{{- end }}
ORDER BY
    table_name
    ,columns
    ,constraint_type
;
