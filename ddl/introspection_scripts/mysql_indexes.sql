SELECT
    table_schema
    ,table_name
    ,index_name
    ,index_type
    ,is_unique
    ,group_concat(column_name ORDER BY seq_in_index) AS columns
    ,group_concat(is_descending ORDER BY seq_in_index) AS descending
FROM (
    SELECT
        table_schema
        ,table_name
        ,index_name
        ,index_type
        ,NOT non_unique AS is_unique
        ,COALESCE(column_name, CONCAT('(', expression, ')')) AS column_name
        ,CASE collation WHEN 'D' THEN 1 ELSE 0 END AS is_descending
        ,seq_in_index
    FROM
        information_schema.statistics
    WHERE
        NOT EXISTS (
            -- exclude indexes created by constraints
            SELECT 1
            FROM information_schema.table_constraints
            WHERE table_constraints.constraint_name = statistics.index_name
        )
        {{- if not .IncludeSystemCatalogs }}
        AND table_schema NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
        {{- end }}
        {{- if .Schemas }}
        AND table_schema IN ({{ mklist .Schemas }})
        {{- else if .ExcludeSchemas }}
        AND table_schema NOT IN ({{ mklist .ExcludeSchemas }})
        {{- end }}
        {{- if .Tables }}
        AND table_name IN ({{ mklist .Tables }})
        {{- else if .ExcludeTables }}
        AND table_name NOT IN ({{ mklist .ExcludeTables }})
        {{- end }}
    ) AS indexed_columns
GROUP BY
    table_schema
    ,table_name
    ,index_name
    ,index_type
    ,is_unique
ORDER BY
    table_schema
    ,table_name
    ,index_name
;
