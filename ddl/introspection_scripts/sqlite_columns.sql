SELECT
    tables.tbl_name AS table_name
    ,columns.name AS column_name
    ,columns.type AS column_type
    ,columns."notnull" AS is_notnull
    ,columns.hidden = 2 AS is_generated
    ,COALESCE(columns.dflt_value, '') AS column_default
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
    CROSS JOIN pragma_table_xinfo(tables.tbl_name) AS columns
ORDER BY
    tables.tbl_name
    ,columns.cid
;
