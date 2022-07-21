SELECT
    table_schema
    ,table_name
    ,COALESCE(table_comment, '') AS table_comment
FROM
    information_schema.tables
WHERE
    table_type = 'BASE TABLE'
    {{- if not .IncludeSystemCatalogs }}
    AND table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
    {{- end }}
    {{- if .Schemas }}
    AND table_schema IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND table_schema NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Tables }}
    AND table_name IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND table_name NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
ORDER BY
    table_schema
    ,table_name
;
