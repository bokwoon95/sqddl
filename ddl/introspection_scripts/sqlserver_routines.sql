SELECT
    schemas.name AS routine_schema
    ,routines.name AS routine_name
    ,CASE routines.type WHEN 'P' THEN 'PROCEDURE' ELSE 'FUNCTION' END AS routine_type
    ,sql_modules.definition AS sql
FROM
    sys.objects AS routines
    JOIN sys.schemas ON schemas.schema_id = routines.schema_id
    JOIN sys.sql_modules ON sql_modules.object_id = routines.object_id
WHERE
    routines.type IN ('P', 'FN', 'IF', 'TF') -- Procedure, Scalar Function, Inline Function, Table Function (https://stackoverflow.com/a/2907204)
    {{- if not .IncludeSystemCatalogs }}
    AND routines.name NOT LIKE 'sp_%'
    {{- end }}
    {{- if .Schemas }}
    AND schemas.name IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND schemas.name NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Routines }}
    AND routines.name IN ({{ mklist .Routines }})
    {{- else if .ExcludeRoutines }}
    AND routines.name NOT IN ({{ mklist .ExcludeRoutines }})
    {{- end }}
ORDER BY
    schemas.name
    ,routines.name
;
