SELECT
    views.tbl_name AS view_name
    ,views.sql || ';' AS sql
    ,group_concat(columns.name, '|') AS column_names
    ,group_concat(columns.type, '|') AS column_types
FROM (
    SELECT
        tbl_name
        ,sql
    FROM
        sqlite_schema
    WHERE
        type = 'view'
        {{- if not .IncludeSystemCatalogs }}
        AND tbl_name NOT LIKE 'sqlite_%' AND sql NOT LIKE 'CREATE TABLE ''%'
        {{- end }}
        {{- if .Views }}
        AND tbl_name IN ({{ mklist .Views }})
        {{- else if .ExcludeViews }}
        AND tbl_name NOT IN ({{ mklist .ExcludeViews }})
        {{- end }}
    ) AS views
    CROSS JOIN pragma_table_xinfo(views.tbl_name) AS columns
GROUP BY
    views.tbl_name
    ,views.sql
ORDER BY
    views.tbl_name
;
