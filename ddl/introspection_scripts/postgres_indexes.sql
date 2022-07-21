SELECT
    table_schema
    ,table_name
    ,index_name
    ,index_type
    ,is_view_index
    ,is_unique
    ,num_key_columns
    ,json_agg(column_name ORDER BY seq) AS columns
    ,json_agg(opclass ORDER BY seq) AS opclasses
    ,COALESCE(pg_get_expr(predicate_oid, table_oid, TRUE), '') AS predicate
    ,pg_get_indexdef(index_oid, 0, TRUE) || ';' AS sql
FROM (
    SELECT
        schemas.nspname AS table_schema
        ,tables.relname AS table_name
        ,indexes.relname AS index_name
        ,pg_am.amname AS index_type
        ,tables.relkind = 'm' AS is_view_index
        ,pg_index.indisunique AS is_unique
        ,pg_index.indnkeyatts AS num_key_columns
        ,pg_get_indexdef(indexes.oid, c.seq::INT, TRUE) AS column_name
        ,pg_opclass.opcname AS opclass
        ,pg_index.indexrelid AS index_oid
        ,pg_index.indrelid AS table_oid
        ,pg_index.indpred AS predicate_oid
        ,c.seq
    FROM
        pg_index
        JOIN pg_class AS indexes ON indexes.oid = pg_index.indexrelid
        JOIN pg_class AS tables ON tables.oid = pg_index.indrelid
        JOIN pg_namespace AS schemas ON schemas.oid = tables.relnamespace
        JOIN pg_am ON pg_am.oid = indexes.relam
        CROSS JOIN unnest(pg_index.indkey) WITH ORDINALITY AS c(oid, seq)
        LEFT JOIN unnest(pg_index.indclass) WITH ORDINALITY AS opclass(oid, seq) ON opclass.seq = c.seq
        LEFT JOIN pg_opclass ON pg_opclass.oid = opclass.oid
        LEFT JOIN pg_attribute AS columns ON columns.attrelid = pg_index.indrelid AND columns.attnum = c.oid
    WHERE
        NOT EXISTS (
            -- exclude indexes created by constraints
            SELECT 1
            FROM pg_constraint
            WHERE pg_constraint.conname = indexes.relname
        )
        {{- if not .IncludeSystemCatalogs }}
        AND schemas.nspname <> 'information_schema' AND schemas.nspname NOT LIKE 'pg_%'
        {{- end }}
        {{- if .Schemas }}
        AND schemas.nspname IN ({{ mklist .Schemas }})
        {{- else if .ExcludeSchemas }}
        AND schemas.nspname NOT IN ({{ mklist .ExcludeSchemas }})
        {{- end }}
        {{- if .Tables }}
        AND tables.relname IN ({{ mklist .Tables }})
        {{- else if .ExcludeTables }}
        AND tables.relname NOT IN ({{ mklist .ExcludeTables }})
        {{- end }}
    ) AS indexed_columns
GROUP BY
    table_schema
    ,table_name
    ,index_name
    ,index_type
    ,is_view_index
    ,is_unique
    ,num_key_columns
    ,index_oid
    ,table_oid
    ,predicate_oid
ORDER BY
    table_schema
    ,table_name
    ,index_name
;
