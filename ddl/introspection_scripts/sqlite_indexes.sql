SELECT
    table_name
    ,index_name
    ,is_unique
    ,group_concat(column_name) AS columns
    ,sql || ';' AS sql
FROM (
    SELECT
        tables.tbl_name AS table_name
        ,indexes.name AS index_name
        ,indexes."unique" AS is_unique
        ,CASE columns.cid
            WHEN -1 THEN '' -- column is the rowid
            WHEN -2 THEN '' -- column is an expression
            ELSE columns.name
        END AS column_name
        ,columns.seqno
        ,m.sql
    FROM (
        SELECT
            tbl_name
        FROM
            sqlite_schema
        WHERE
            type = 'table'
            {{- if not .IncludeSystemCatalogs }}
            AND tbl_name NOT LIKE 'sqlite_%' AND sql NOT LIKE 'CREATE TABLE ''%'
            {{- end }}
            {{- if .Tables }}
            AND tbl_name IN ({{ mklist .Tables }})
            {{- else if .ExcludeTables }}
            AND tbl_name NOT IN ({{ mklist .ExcludeTables }})
            {{- end }}
        ) AS tables
        CROSS JOIN pragma_index_list(tables.tbl_name) AS indexes
        CROSS JOIN pragma_index_info(indexes.name) AS columns
        JOIN sqlite_schema AS m ON m.type = 'index' AND m.tbl_name = tables.tbl_name AND m.name = indexes.name
    WHERE
        indexes.origin = 'c' -- 'c' = 'CREATE INDEX', 'u' = 'UNIQUE', 'pk' = 'PRIMARY KEY'
    ORDER BY
        indexes.name
        ,columns.seqno
    ) AS index_columns
GROUP BY
    table_name
    ,index_name
    ,is_unique
    ,sql
ORDER BY
    table_name
    ,index_name
;
