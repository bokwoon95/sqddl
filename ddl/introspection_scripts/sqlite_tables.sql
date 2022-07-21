SELECT
    m.tbl_name AS table_name
    ,m.sql || ';' AS sql
FROM
    sqlite_schema AS m
WHERE
    type = 'table'
    {{- if not .IncludeSystemCatalogs }}
    {{- if .VersionNums.LowerThan 3 37 }}
    AND m.tbl_name NOT LIKE 'sqlite_%' AND m.sql NOT LIKE 'CREATE TABLE ''%'
    {{- else }}
    AND m.tbl_name NOT LIKE 'sqlite_%' AND EXISTS (
        SELECT 1
        FROM pragma_table_list AS tl
        WHERE tl."type" IN ('table', 'virtual') AND tl.schema = 'main' AND tl.name = m.tbl_name
    )
    {{- end }}
    {{- end }}
    {{- if .Tables }}
    AND m.tbl_name IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND m.tbl_name NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
ORDER BY
    m.tbl_name
;
