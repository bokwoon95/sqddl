SELECT
    tbl_name AS table_name
    ,name AS trigger_name
    ,sql || ';' AS sql
FROM
    sqlite_schema
WHERE
    type = 'trigger'
    {{- if not .IncludeSystemCatalogs }}
    AND tbl_name NOT LIKE 'sqlite_%' AND sql NOT LIKE 'CREATE TABLE ''%'
    {{- end }}
    {{- if .Tables }}
    AND tbl_name IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND tbl_name NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
ORDER BY
    tbl_name
    ,name
;
