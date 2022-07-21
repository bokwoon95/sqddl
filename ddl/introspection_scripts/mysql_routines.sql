SELECT
    routines.routine_schema
    ,routines.routine_name
    ,routines.routine_type
    ,COALESCE(routines.dtd_identifier, '') AS return_type
    ,parameters.parameters
    ,routines.routine_definition AS `sql`
    ,routines.routine_comment
    ,routines.routine_body
    ,routines.is_deterministic = 'YES' AS is_deterministic
    ,routines.sql_data_access
    ,routines.security_type
FROM
    information_schema.routines
    JOIN (
        SELECT
            specific_schema
            ,specific_name
            ,group_concat(
                CASE routine_type WHEN 'FUNCTION' THEN '' ELSE CONCAT(parameter_mode, ' ') END,
                CONCAT(parameter_name, ' '),
                dtd_identifier
                ORDER BY ordinal_position
                SEPARATOR '|'
            ) AS parameters
        FROM
            information_schema.parameters
        GROUP BY
            specific_schema
            ,specific_name
    ) AS parameters ON parameters.specific_schema = routines.routine_schema AND parameters.specific_name = routines.routine_name
WHERE
    TRUE
    {{- if not .IncludeSystemCatalogs }}
    AND routines.routine_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
    {{- end }}
    {{- if .Schemas }}
    AND routines.routine_schema IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND routines.routine_schema NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Routines }}
    AND routines.routine_name IN ({{ mklist .Routines }})
    {{- else if .ExcludeRoutines }}
    AND routines.routine_name NOT IN ({{ mklist .ExcludeRoutines }})
    {{- end }}
ORDER BY
    routines.routine_schema
    ,routines.routine_name
;
