SELECT
    schemas.name AS table_schema
    ,tables.name AS table_name
    ,triggers.name AS trigger_name
    ,CASE WHEN tables.type = 'V' THEN 1 ELSE 0 END AS is_view_trigger
    ,sql_modules.definition AS sql
FROM
    sys.objects AS triggers
    JOIN sys.objects AS tables ON tables.object_id = triggers.parent_object_id
    JOIN sys.schemas ON schemas.schema_id = tables.schema_id
    JOIN sys.sql_modules ON sql_modules.object_id = triggers.object_id
WHERE
    triggers.type = 'TR' -- Trigger (https://stackoverflow.com/a/2907204)
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
    {{- if .Views }}
    AND tables.name IN ({{ mklist .Views }})
    {{- else if .ExcludeViews }}
    AND tables.name NOT IN ({{ mklist .ExcludeViews }})
    {{- end }}
ORDER BY
    schemas.name
    ,tables.name
    ,triggers.name
;
