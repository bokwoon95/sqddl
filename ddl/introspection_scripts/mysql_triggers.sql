SELECT
    event_object_schema AS table_schema
    ,event_object_table AS table_name
    ,trigger_name
    ,action_statement AS `sql`
    ,action_timing
    ,event_manipulation
FROM
    information_schema.triggers
WHERE
    TRUE
    {{- if not .IncludeSystemCatalogs }}
    AND event_object_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
    {{- end }}
    {{- if .Schemas }}
    AND event_object_schema IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND event_object_schema NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Tables }}
    AND event_object_table IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND event_object_table NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
ORDER BY
    event_object_schema
    ,event_object_table
    ,trigger_name
;
