SELECT
    table_schema AS view_schema
    ,table_name AS view_name
    ,view_definition AS `sql`
    ,columns.names AS column_names
    ,columns.types AS column_types
FROM
    information_schema.views
    JOIN (
        SELECT
            table_schema
            ,table_name
            ,group_concat(column_name ORDER BY ordinal_position SEPARATOR '|') AS names
            ,group_concat(column_type ORDER BY ordinal_position SEPARATOR '|') AS types
        FROM information_schema.columns
        GROUP BY table_schema, table_name
    ) AS columns USING (table_schema, table_name)
WHERE
    TRUE
    {{- if not .IncludeSystemCatalogs }}
    AND table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
    {{- end }}
    {{- if .Schemas }}
    AND table_schema IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND table_schema NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Views }}
    AND table_name IN ({{ mklist .Views }})
    {{- else if .ExcludeViews }}
    AND table_name NOT IN ({{ mklist .ExcludeViews }})
    {{- end }}
ORDER BY
    table_schema
    ,table_name
;
