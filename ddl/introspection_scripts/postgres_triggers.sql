SELECT
    schemas.nspname AS table_schema
    ,tables.relname AS table_name
    ,pg_trigger.tgname AS trigger_name
    ,tables.relkind = 'v' AS is_view_trigger
    ,COALESCE(pg_get_triggerdef(pg_trigger.oid, TRUE) || ';', '') AS sql
    ,COALESCE(pg_description.description, '') AS trigger_comment
FROM
    pg_trigger
    JOIN pg_class AS tables ON tables.oid = pg_trigger.tgrelid
    JOIN pg_namespace AS schemas ON schemas.oid = tables.relnamespace
    LEFT JOIN pg_description ON pg_description.objoid = pg_trigger.oid
WHERE
    NOT pg_trigger.tgisinternal -- exclude system triggers
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
    ,pg_trigger.tgname
;
