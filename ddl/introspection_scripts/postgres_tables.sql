SELECT
    schemas.nspname AS table_schema
    ,tables.relname AS table_name
    ,COALESCE(pg_description.description, '') AS table_comment
FROM
    pg_class AS tables
    JOIN pg_namespace AS schemas ON schemas.oid = tables.relnamespace
    LEFT JOIN pg_description ON pg_description.objoid = tables.oid
WHERE
    tables.relkind = 'r'
    {{- if not .IncludeSystemCatalogs }}
    AND schemas.nspname <> 'information_schema' AND schemas.nspname NOT LIKE 'pg_%'
    {{- end }}
    {{- if .Schemas }}
    AND schemas.nspname IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND schemas.nspname NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Tables }}
    AND tables.relname IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND tables.relname NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
ORDER BY
    schemas.nspname
    ,tables.relname
;
