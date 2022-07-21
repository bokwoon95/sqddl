SELECT
    schemas.name AS table_schema
    ,views.name AS view_name
    ,sql_modules.definition AS sql
    ,columns.names AS column_names
    ,columns.types AS column_types
FROM
    sys.objects AS views
    JOIN sys.schemas ON schemas.schema_id = views.schema_id
    JOIN sys.sql_modules ON sql_modules.object_id = views.object_id
    JOIN (
        SELECT
            object_id
            ,string_agg(name, '|') WITHIN GROUP (ORDER BY column_id) AS names
            ,string_agg(TYPE_NAME(user_type_id), '|') WITHIN GROUP (ORDER BY column_id) AS types
        FROM sys.columns
        GROUP BY object_id
    ) AS columns ON columns.object_id = views.object_id
WHERE
    views.type = 'V' -- View (https://stackoverflow.com/a/2907204)
    {{- if not .IncludeSystemCatalogs }}
    AND views.name NOT LIKE 'spt_%' AND views.name NOT LIKE 'msreplication_%'
    {{- end }}
    {{- if .Schemas }}
    AND schemas.name IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND schemas.name NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Views }}
    AND views.name IN ({{ mklist .Views }})
    {{- else if .ExcludeViews }}
    AND views.name NOT IN ({{ mklist .ExcludeViews }})
    {{- end }}
ORDER BY
    schemas.name
    ,views.name
;
