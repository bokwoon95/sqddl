SELECT
    schemas.nspname AS enum_schema
    ,pg_type.typname AS enum_name
    ,json_agg(pg_enum.enumlabel::TEXT ORDER BY pg_enum.enumsortorder) AS enum_labels
    ,COALESCE(obj_description(pg_type.oid, current_database()), '') AS enum_comment
FROM
    pg_enum
    JOIN pg_type ON pg_type.oid = pg_enum.enumtypid
    JOIN pg_namespace AS schemas ON schemas.oid = pg_type.typnamespace
WHERE
    TRUE
    {{- if not .IncludeSystemCatalogs }}
    AND schemas.nspname <> 'information_schema' AND schemas.nspname NOT LIKE 'pg_%'
    {{- end }}
    {{- if .Schemas }}
    AND schemas.nspname IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND schemas.nspname NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Enums }}
    AND pg_typ.typname IN ({{ mklist .Enums }})
    {{- else if .ExcludeEnums }}
    AND pg_typ.typname NOT IN ({{ mklist .ExcludeEnums }})
    {{- end }}
GROUP BY
    schemas.nspname
    ,pg_type.typname
    ,pg_type.oid
ORDER BY
    schemas.nspname
    ,pg_type.typname
;
