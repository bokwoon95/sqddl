SELECT
    schemas.name AS table_schema
    ,tables.name AS table_name
FROM
    sys.objects AS tables
    JOIN sys.schemas ON schemas.schema_id = tables.schema_id
WHERE
    tables.type = 'U' -- User-defined table (https://stackoverflow.com/a/2907204)
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
ORDER BY
    schemas.name
    ,tables.name
;
