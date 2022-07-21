SELECT
    '' AS table_schema
    ,'' AS table_name
    ,'' AS constraint_name
    ,'' AS constraint_type
    ,'[]'::JSON AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'[]'::JSON AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS match_option
    ,'' AS check_expr
    ,'[]'::JSON AS operators
    ,'' AS index_type
    ,'' AS predicate
    ,FALSE AS is_deferrable
    ,FALSE AS is_initially_deferred
    ,FALSE AS is_not_valid
WHERE
    FALSE
{{- if .IncludeConstraintType "PRIMARY KEY" }}
UNION ALL
SELECT
    table_schema
    ,table_name
    ,constraint_name
    ,'PRIMARY KEY' AS constraint_type
    ,json_agg(column_name ORDER BY seq) AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'[]'::JSON AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS match_option
    ,'' AS check_expr
    ,'[]'::JSON AS operators
    ,'' AS index_type
    ,'' AS predicate
    ,is_deferrable
    ,is_initially_deferred
    ,FALSE AS is_not_valid
FROM (
    SELECT
        schemas.nspname AS table_schema
        ,tables.relname AS table_name
        ,pg_constraint.conname AS constraint_name
        ,columns.attname AS column_name
        ,pg_constraint.condeferrable AS is_deferrable
        ,pg_constraint.condeferred AS is_initially_deferred
        ,c.seq
    FROM
        pg_constraint
        JOIN pg_class AS tables ON tables.oid = pg_constraint.conrelid
        JOIN pg_namespace AS schemas ON schemas.oid = tables.relnamespace
        CROSS JOIN unnest(pg_constraint.conkey) WITH ORDINALITY AS c(oid, seq)
        JOIN pg_attribute AS columns ON columns.attrelid = pg_constraint.conrelid AND columns.attnum = c.oid
    WHERE
        pg_constraint.contype = 'p'
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
) AS primary_key_columns
GROUP BY
    table_schema
    ,table_name
    ,constraint_name
    ,is_deferrable
    ,is_initially_deferred
{{- end }}
{{- if .IncludeConstraintType "UNIQUE" }}
UNION ALL
SELECT
    table_schema
    ,table_name
    ,constraint_name
    ,'UNIQUE' AS constraint_type
    ,json_agg(column_name ORDER BY seq) AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'[]'::JSON AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS match_option
    ,'' AS check_expr
    ,'[]'::JSON AS operators
    ,'' AS index_type
    ,'' AS predicate
    ,is_deferrable
    ,is_initially_deferred
    ,FALSE AS is_not_valid
FROM (
    SELECT
        schemas.nspname AS table_schema
        ,tables.relname AS table_name
        ,pg_constraint.conname AS constraint_name
        ,columns.attname AS column_name
        ,pg_constraint.condeferrable AS is_deferrable
        ,pg_constraint.condeferred AS is_initially_deferred
        ,c.seq
    FROM
        pg_constraint
        JOIN pg_class AS tables ON tables.oid = pg_constraint.conrelid
        JOIN pg_namespace AS schemas ON schemas.oid = tables.relnamespace
        CROSS JOIN unnest(pg_constraint.conkey) WITH ORDINALITY AS c(oid, seq)
        JOIN pg_attribute AS columns ON columns.attrelid = pg_constraint.conrelid AND columns.attnum = c.oid
    WHERE
        pg_constraint.contype = 'u'
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
) AS unique_columns
GROUP BY
    table_schema
    ,table_name
    ,constraint_name
    ,is_deferrable
    ,is_initially_deferred
{{- end }}
{{- if .IncludeConstraintType "FOREIGN KEY" }}
UNION ALL
SELECT
    table_schema
    ,table_name
    ,constraint_name
    ,'FOREIGN KEY' AS constraint_type
    ,json_agg(column_name ORDER BY seq) AS columns
    ,references_schema
    ,references_table
    ,json_agg(references_column ORDER BY seq) AS references_columns
    ,update_rule
    ,delete_rule
    ,match_option
    ,'' AS check_expr
    ,'[]'::JSON AS operators
    ,'' AS index_type
    ,'' AS predicate
    ,is_deferrable
    ,is_initially_deferred
    ,is_not_valid
FROM (
    SELECT
        schemas1.nspname AS table_schema
        ,tables1.relname AS table_name
        ,pg_constraint.conname AS constraint_name
        ,columns1.attname AS column_name
        ,schemas2.nspname AS references_schema
        ,tables2.relname AS references_table
        ,columns2.attname AS references_column
        ,CASE pg_constraint.confupdtype
            WHEN 'a' THEN 'NO ACTION'
            WHEN 'r' THEN 'RESTRICT'
            WHEN 'c' THEN 'CASCADE'
            WHEN 'n' THEN 'SET NULL'
            WHEN 'd' THEN 'SET DEFAULT'
        END AS update_rule
        ,CASE pg_constraint.confdeltype
            WHEN 'a' THEN 'NO ACTION'
            WHEN 'r' THEN 'RESTRICT'
            WHEN 'c' THEN 'CASCADE'
            WHEN 'n' THEN 'SET NULL'
            WHEN 'd' THEN 'SET DEFAULT'
        END AS delete_rule
        ,CASE pg_constraint.confmatchtype
            WHEN 'f' THEN 'MATCH FULL'
            WHEN 'p' THEN 'MATCH PARTIAL'
            ELSE ''
        END AS match_option
        ,pg_constraint.condeferrable AS is_deferrable
        ,pg_constraint.condeferred AS is_initially_deferred
        ,NOT pg_constraint.convalidated AS is_not_valid
        ,c1.seq
    FROM
        pg_constraint
        JOIN pg_class AS tables1 ON tables1.oid = pg_constraint.conrelid
        JOIN pg_class AS tables2 ON tables2.oid = pg_constraint.confrelid
        JOIN pg_namespace AS schemas1 ON schemas1.oid = tables1.relnamespace
        JOIN pg_namespace AS schemas2 ON schemas2.oid = tables2.relnamespace
        CROSS JOIN unnest(pg_constraint.conkey) WITH ORDINALITY AS c1(oid, seq)
        JOIN unnest(pg_constraint.confkey) WITH ORDINALITY AS c2(oid, seq) ON c2.seq = c1.seq
        JOIN pg_attribute AS columns1 ON columns1.attrelid = pg_constraint.conrelid AND columns1.attnum = c1.oid
        JOIN pg_attribute AS columns2 ON columns2.attrelid = pg_constraint.confrelid AND columns2.attnum = c2.oid
    WHERE
        pg_constraint.contype = 'f'
        {{- if not .IncludeSystemCatalogs }}
        AND schemas1.nspname <> 'information_schema' AND schemas1.nspname NOT LIKE 'pg_%'
        {{- end }}
        {{- if .Schemas }}
        AND schemas1.nspname IN ({{ mklist .Schemas }})
        {{- else if .ExcludeSchemas }}
        AND schemas1.nspname NOT IN ({{ mklist .ExcludeSchemas }})
        {{- end }}
        {{- if .Tables }}
        AND tables1.relname IN ({{ mklist .Tables }})
        {{- else if .ExcludeTables }}
        AND tables1.relname NOT IN ({{ mklist .ExcludeTables }})
        {{- end }}
) AS foreign_key_columns
GROUP BY
    table_schema
    ,table_name
    ,constraint_name
    ,references_schema
    ,references_table
    ,update_rule
    ,delete_rule
    ,match_option
    ,is_deferrable
    ,is_initially_deferred
    ,is_not_valid
{{- end }}
{{- if .IncludeConstraintType "CHECK" }}
UNION ALL
SELECT
    schemas.nspname AS table_schema
    ,tables.relname AS table_name
    ,pg_constraint.conname AS constraint_name
    ,'CHECK' AS constraint_type
    ,'[]'::JSON AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'[]'::JSON AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS match_option
    ,pg_get_constraintdef(pg_constraint.oid, TRUE) AS check_expr
    ,'[]'::JSON AS operators
    ,'' AS index_type
    ,'' AS predicate
    ,pg_constraint.condeferrable AS is_deferrable
    ,pg_constraint.condeferred AS is_initially_deferred
    ,NOT pg_constraint.convalidated AS is_not_valid
FROM
    pg_constraint
    JOIN pg_class AS tables ON tables.oid = pg_constraint.conrelid
    JOIN pg_namespace AS schemas ON schemas.oid = tables.relnamespace
WHERE
    pg_constraint.contype = 'c'
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
{{- end }}
{{- if .IncludeConstraintType "EXCLUDE" }}
UNION ALL
SELECT
    table_schema
    ,table_name
    ,constraint_name
    ,'EXCLUDE' AS constraint_type
    ,json_agg(column_name ORDER BY seq) AS columns
    ,'' AS references_schema
    ,'' AS references_table
    ,'[]'::JSON AS references_columns
    ,'' AS update_rule
    ,'' AS delete_rule
    ,'' AS match_option
    ,'' AS check_expr
    ,json_agg(operator ORDER BY seq) AS operators
    ,exclusion_index_type
    ,COALESCE(pg_get_expr(predicate_oid, table_oid, TRUE), '') AS predicate
    ,is_deferrable
    ,is_initially_deferred
    ,FALSE AS is_not_valid
FROM (
    SELECT
        schemas.nspname AS table_schema
        ,tables.relname AS table_name
        ,pg_constraint.conname AS constraint_name
        ,pg_get_indexdef(indexes.oid, c.seq::INT, TRUE) AS column_name
        ,pg_operator.oprname AS operator
        ,pg_am.amname AS exclusion_index_type
        ,pg_constraint.condeferrable AS is_deferrable
        ,pg_constraint.condeferred AS is_initially_deferred
        ,pg_index.indrelid AS table_oid
        ,pg_index.indpred AS predicate_oid
        ,c.seq
    FROM
        pg_constraint
        JOIN pg_class AS tables ON tables.oid = pg_constraint.conrelid
        JOIN pg_class AS indexes ON indexes.oid = pg_constraint.conindid
        JOIN pg_namespace AS schemas ON schemas.oid = tables.relnamespace
        JOIN pg_index ON pg_index.indexrelid = indexes.oid
        JOIN pg_am ON pg_am.oid = indexes.relam
        CROSS JOIN unnest(pg_index.indkey) WITH ORDINALITY AS c(oid, seq)
        JOIN unnest(pg_constraint.conexclop) WITH ORDINALITY AS o(oid, seq) ON o.seq = c.seq
        LEFT JOIN pg_attribute AS columns ON columns.attrelid = pg_index.indrelid AND columns.attnum = c.oid
        JOIN pg_operator ON pg_operator.oid = o.oid
    WHERE
        pg_constraint.contype = 'x'
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
) AS exclude_columns
GROUP BY
    table_schema
    ,table_name
    ,constraint_name
    ,exclusion_index_type
    ,is_deferrable
    ,is_initially_deferred
    ,table_oid
    ,predicate_oid
{{- end }}
ORDER BY
    table_schema
    ,table_name
    ,constraint_name
;
