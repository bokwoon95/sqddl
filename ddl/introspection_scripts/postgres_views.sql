SELECT
    pg_views.schemaname AS view_schema
    ,pg_views.viewname AS view_name
    ,FALSE AS is_materialized
    ,pg_get_viewdef(to_regclass(pg_views.schemaname || '.' || pg_views.viewname), TRUE) AS sql
    ,columns.names AS column_names
    ,columns.types AS column_types
    ,columns.enums AS column_enums
    ,COALESCE(obj_description(to_regclass(pg_views.schemaname || '.' || pg_views.viewname), current_database()), '') AS view_comment
FROM
    pg_views
    JOIN (
        SELECT
            pg_attribute.attrelid
            ,string_agg(pg_attribute.attname, '|' ORDER BY pg_attribute.attnum) AS names
            ,string_agg(UPPER(format_type(pg_attribute.atttypid, pg_attribute.atttypmod)), '|' ORDER BY pg_attribute.attnum) AS types
            ,string_agg(CAST(enum.oid IS NOT NULL AS TEXT), '|' ORDER BY pg_attribute.attnum) AS enums
        FROM
            pg_attribute
            LEFT JOIN pg_type AS enum ON enum.typtype = 'e' AND enum.oid = pg_attribute.atttypid
        WHERE
            pg_attribute.attnum > 0           -- exclude system columns
            AND NOT pg_attribute.attisdropped -- exclude dropped columns
        GROUP BY
            pg_attribute.attrelid
    ) AS columns ON columns.attrelid = to_regclass(pg_views.schemaname || '.' || pg_views.viewname)
WHERE
    TRUE
    {{- if not .IncludeSystemCatalogs }}
    AND pg_views.schemaname <> 'information_schema' AND pg_views.schemaname NOT LIKE 'pg_%'
    {{- end }}
    {{- if .Schemas }}
    AND pg_views.schemaname IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND pg_views.schemaname NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Views }}
    AND pg_views.viewname IN ({{ mklist .Views }})
    {{- else if .ExcludeViews }}
    AND pg_views.viewname NOT IN ({{ mklist .ExcludeViews }})
    {{- end }}
UNION ALL
SELECT
    pg_matviews.schemaname AS view_schema
    ,pg_matviews.matviewname AS view_name
    ,TRUE AS is_materialized
    ,pg_get_viewdef(to_regclass(pg_matviews.schemaname || '.' || pg_matviews.matviewname), TRUE) AS sql
    ,columns.names AS column_names
    ,columns.types AS column_types
    ,columns.enums AS column_enums
    ,COALESCE(obj_description(to_regclass(pg_matviews.schemaname || '.' || pg_matviews.matviewname), current_database()), '') AS view_comment
FROM
    pg_matviews
    JOIN (
        SELECT
            pg_attribute.attrelid
            ,string_agg(pg_attribute.attname, '|' ORDER BY pg_attribute.attnum) AS names
            ,string_agg(UPPER(format_type(pg_attribute.atttypid, pg_attribute.atttypmod)), '|' ORDER BY pg_attribute.attnum) AS types
            ,string_agg(CAST(enum.oid IS NOT NULL AS TEXT), '|' ORDER BY pg_attribute.attnum) AS enums
        FROM
            pg_attribute
            LEFT JOIN pg_type AS enum ON enum.typtype = 'e' AND enum.oid = pg_attribute.atttypid
        WHERE
            pg_attribute.attnum > 0           -- exclude system columns
            AND NOT pg_attribute.attisdropped -- exclude dropped columns
        GROUP BY
            pg_attribute.attrelid
    ) AS columns ON columns.attrelid = to_regclass(pg_matviews.schemaname || '.' || pg_matviews.matviewname)
WHERE
    TRUE
    {{- if not .IncludeSystemCatalogs }}
    AND pg_matviews.schemaname <> 'information_schema' AND pg_matviews.schemaname NOT LIKE 'pg_%'
    {{- end }}
    {{- if .Schemas }}
    AND pg_matviews.schemaname IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND pg_matviews.schemaname NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Views }}
    AND pg_matviews.matviewname IN ({{ mklist .Views }})
    {{- else if .ExcludeViews }}
    AND pg_matviews.matviewname NOT IN ({{ mklist .ExcludeViews }})
    {{- end }}
ORDER BY
    view_schema
    ,view_name
;
