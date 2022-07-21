SELECT
    schemas.nspname AS routine_schema
    ,pg_proc.proname AS routine_name
    {{- if .VersionNums.LowerThan 11 }}
    -- https://dba.stackexchange.com/a/238906
    ,CASE
        WHEN pg_proc.proisagg THEN 'AGGREGATE'
        WHEN pg_proc.proiswindow THEN 'WINDOW'
        ELSE 'FUNCTION'
    END AS routine_type
    {{- else }}
    ,CASE pg_proc.prokind
        WHEN 'a' THEN 'AGGREGATE'
        WHEN 'w' THEN 'WINDOW'
        WHEN 'p' THEN 'PROCEDURE'
        ELSE 'FUNCTION'
    END AS routine_type
    {{- end }}
    ,pg_get_function_identity_arguments(pg_proc.oid) AS identity_arguments
    ,pg_get_functiondef(pg_proc.oid) || ';' AS sql
    ,COALESCE(pg_get_function_result(pg_proc.oid), '') AS return_type
    ,COALESCE(pg_description.description, '') AS routine_comment
FROM
    pg_proc
    JOIN pg_namespace AS schemas ON schemas.oid = pg_proc.pronamespace
    LEFT JOIN pg_description ON pg_description.objoid = pg_proc.oid
WHERE
    NOT EXISTS (
        -- exclude functions created by extensions
        SELECT
            1
        FROM
            pg_extension
            JOIN pg_depend ON pg_depend.refobjid = pg_extension.oid
            JOIN pg_proc AS proc2 ON proc2.oid = pg_depend.objid
        WHERE
            pg_depend.deptype = 'e'
            AND pg_proc.oid = proc2.oid
    )
    {{- if not .IncludeSystemCatalogs }}
    AND schemas.nspname <> 'information_schema' AND schemas.nspname NOT LIKE 'pg_%'
    {{- end }}
    {{- if .Schemas }}
    AND schemas.nspname IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND schemas.nspname NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Routines }}
    AND pg_proc.proname IN ({{ mklist .Routines }})
    {{- else if .ExcludeRoutines }}
    AND pg_proc.proname NOT IN ({{ mklist .ExcludeRoutines }})
    {{- end }}
ORDER BY
    schemas.nspname
    ,pg_proc.proname
;
