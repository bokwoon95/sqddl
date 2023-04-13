SELECT
    '' AS table_schema
    ,'' AS table_name
    ,'' AS constraint_name
    ,'' AS constraint_type
    ,'' AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS match_option
    ,'' AS check_expr
FROM
    information_schema.table_constraints
WHERE
    FALSE
{{- if .IncludeConstraintType "PRIMARY KEY" }}
UNION ALL
SELECT
    tc.table_schema
    ,tc.table_name
    ,tc.constraint_name
    ,'PRIMARY KEY' AS constraint_type
    ,group_concat(kcu.column_name ORDER BY kcu.ordinal_position) AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS match_option
    ,'' AS check_expr
FROM
    information_schema.table_constraints AS tc
    JOIN information_schema.key_column_usage AS kcu USING (constraint_schema, constraint_name, table_name)
WHERE
    tc.constraint_type = 'PRIMARY KEY'
    {{- if not .IncludeSystemCatalogs }}
    AND tc.table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
    {{- end }}
    {{- if .Schemas }}
    AND tc.table_schema IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND tc.table_schema NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Tables }}
    AND tc.table_name IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND tc.table_name NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
GROUP BY
    tc.table_schema
    ,tc.table_name
    ,tc.constraint_name
    ,tc.constraint_type
{{- end }}
{{- if .IncludeConstraintType "UNIQUE" }}
UNION ALL
SELECT
    tc.table_schema
    ,tc.table_name
    ,tc.constraint_name
    ,'UNIQUE' AS constraint_type
    ,group_concat(kcu.column_name ORDER BY kcu.ordinal_position) AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS match_option
    ,'' AS check_expr
FROM
    information_schema.table_constraints AS tc
    JOIN information_schema.key_column_usage AS kcu USING (constraint_schema, constraint_name, table_name)
WHERE
    tc.constraint_type = 'UNIQUE'
    {{- if not .IncludeSystemCatalogs }}
    AND tc.table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
    {{- end }}
    {{- if .Schemas }}
    AND tc.table_schema IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND tc.table_schema NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Tables }}
    AND tc.table_name IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND tc.table_name NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
GROUP BY
    tc.table_schema
    ,tc.table_name
    ,tc.constraint_name
    ,tc.constraint_type
{{- end }}
{{- if .IncludeConstraintType "FOREIGN KEY" }}
UNION ALL
SELECT
    tc.table_schema
    ,tc.table_name
    ,tc.constraint_name
    ,tc.constraint_type
    ,group_concat(kcu.column_name ORDER BY kcu.ordinal_position) AS columns
    ,COALESCE(kcu.referenced_table_schema, '') AS references_schema
    ,COALESCE(kcu.referenced_table_name, '') AS references_table
    ,COALESCE(group_concat(kcu.referenced_column_name ORDER BY kcu.ordinal_position), '') AS references_columns
    ,COALESCE(rc.update_rule, '') AS update_rule
    ,COALESCE(rc.delete_rule, '') AS delete_rule
    ,CASE rc.match_option WHEN 'NONE' THEN '' ELSE COALESCE(CONCAT('MATCH ', rc.match_option), '') END AS match_option
    ,'' AS check_expr
FROM
    information_schema.table_constraints AS tc
    JOIN information_schema.key_column_usage AS kcu USING (constraint_schema, constraint_name, table_name)
    LEFT JOIN information_schema.referential_constraints AS rc USING (constraint_schema, constraint_name)
WHERE
    tc.constraint_type = 'FOREIGN KEY'
    {{- if not .IncludeSystemCatalogs }}
    AND tc.table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
    {{- end }}
    {{- if .Schemas }}
    AND tc.table_schema IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND tc.table_schema NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Tables }}
    AND tc.table_name IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND tc.table_name NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
GROUP BY
    tc.table_schema
    ,tc.table_name
    ,tc.constraint_name
    ,tc.constraint_type
    ,kcu.referenced_table_schema
    ,kcu.referenced_table_name
    ,rc.update_rule
    ,rc.delete_rule
    ,rc.match_option
{{- end }}
{{- if and (.IncludeConstraintType "CHECK") (.VersionNums.GreaterOrEqualTo 8) }}
UNION ALL
SELECT
    tc.table_schema
    ,tc.table_name
    ,tc.constraint_name
    ,'CHECK' AS constrain_type
    ,'' AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'' AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS match_option
    ,cc.check_clause AS check_expr
FROM
    information_schema.table_constraints AS tc
    JOIN information_schema.check_constraints AS cc USING (constraint_schema, constraint_name)
WHERE
    TRUE
    {{- if not .IncludeSystemCatalogs }}
    AND tc.table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
    {{- end }}
    {{- if .Schemas }}
    AND tc.table_schema IN ({{ mklist .Schemas }})
    {{- else if .ExcludeSchemas }}
    AND tc.table_schema NOT IN ({{ mklist .ExcludeSchemas }})
    {{- end }}
    {{- if .Tables }}
    AND tc.table_name IN ({{ mklist .Tables }})
    {{- else if .ExcludeTables }}
    AND tc.table_name NOT IN ({{ mklist .ExcludeTables }})
    {{- end }}
{{- end }}
ORDER BY
    table_schema
    ,table_name
    ,constraint_name
;
