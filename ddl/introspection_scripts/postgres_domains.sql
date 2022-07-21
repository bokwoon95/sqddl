SELECT
    schemas.nspname AS domain_schema
    ,pg_type.typname AS domain_name
    ,pg_catalog.format_type(pg_type.typbasetype, pg_type.typtypmod) AS underlying_type
    ,COALESCE(pg_collation.collname, '') AS collation
    ,pg_type.typnotnull AS is_notnull
    ,COALESCE(pg_type.typdefault, '') AS column_default
    ,json_agg(COALESCE(pg_constraint.conname, '') ORDER BY pg_constraint.oid) AS check_names
    ,json_agg(COALESCE(pg_get_constraintdef(pg_constraint.oid, true), '') ORDER BY pg_constraint.oid) AS check_exprs
    ,COALESCE(obj_description(pg_type.oid, current_database()), '') AS domain_comment
FROM
    pg_type
    LEFT JOIN pg_namespace AS schemas ON schemas.oid = pg_type.typnamespace
    LEFT JOIN pg_type AS base_type ON base_type.oid = pg_type.typbasetype
    LEFT JOIN pg_collation ON pg_collation.oid = pg_type.typcollation AND pg_type.typcollation <> base_type.typcollation
    LEFT JOIN pg_constraint ON pg_constraint.contypid = pg_type.oid
WHERE
    pg_type.typtype = 'd'
    {{- if not .IncludeSystemCatalogs }}
    AND schemas.nspname <> 'information_schema' AND schemas.nspname NOT LIKE 'pg_%'
    {{- end }}
    {{- if .Schemas }}
    AND schemas.nspname IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND schemas.nspname NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Domains }}
    pg_type.typname IN ({{ mklist .Domains }})
    {{- else if .ExcludeDomains }}
    pg_type.typname NOT IN ({{ mklist .ExcludeDomains }})
    {{- end }}
GROUP BY
    schemas.nspname
    ,pg_type.typname
    ,pg_type.typbasetype
    ,pg_type.typtypmod
    ,pg_collation.collname
    ,pg_type.typnotnull
    ,pg_type.typdefault
    ,pg_type.oid
ORDER BY
    domain_schema
    ,domain_name
;
