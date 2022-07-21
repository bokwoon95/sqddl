SELECT
    table_schema
    ,table_name
    ,index_name
    ,index_type
    ,is_view_index
    ,is_unique
    ,string_agg(column_name, ',') WITHIN GROUP (ORDER BY key_ordinal) AS columns
    ,string_agg(CAST(is_descending_key AS CHAR(1)), ',') WITHIN GROUP (ORDER BY key_ordinal) AS descending
    ,string_agg(CAST(is_included_column AS CHAR(1)), ',') WITHIN GROUP (ORDER BY key_ordinal) AS included
    ,predicate
FROM (
    SELECT
        schemas.name AS table_schema
        ,tables.name AS table_name
        ,indexes.name AS index_name
        ,indexes.type_desc AS index_type
        ,0 AS is_view_index
        ,indexes.is_unique
        ,columns.name AS column_name
        ,index_columns.is_descending_key
        ,index_columns.is_included_column
        ,COALESCE(indexes.filter_definition, '') AS predicate
        ,index_columns.key_ordinal
    FROM
        sys.indexes
        JOIN sys.tables ON tables.object_id = indexes.object_id
        JOIN sys.schemas ON schemas.schema_id = tables.schema_id
        JOIN sys.index_columns
            ON index_columns.index_id = indexes.index_id
            AND index_columns.object_id = indexes.object_id
        JOIN sys.columns
            ON columns.column_id = index_columns.column_id
            AND columns.object_id = indexes.object_id
    WHERE
        indexes.name IS NOT NULL
        AND indexes.is_primary_key = 0 AND indexes.is_unique_constraint = 0
        {{- if not .IncludeSystemCatalogs }}
        AND tables.name NOT LIKE 'spt_%' AND tables.name NOT LIKE 'msreplication_%'
        {{- end }}
        {{- if .Schemas }}
        AND schemas.name IN ({{ mklist .Schemas }})
        {{- else if .ExcludeSchemas }}
        AND schemas.name NOT IN ({{ mklist .ExcludeSchemas }})
        {{- end }}
        {{- if .Tables }}
        AND tables.name IN ({{ mklist .Tables }})
        {{- else if .ExcludeTables }}
        AND tables.name NOT IN ({{ mklist .ExcludeTables }})
        {{- end }}
    UNION ALL
    SELECT
        schemas.name AS table_schema
        ,views.name AS table_name
        ,indexes.name AS index_name
        ,indexes.type_desc AS index_type
        ,1 AS is_view_index
        ,indexes.is_unique
        ,columns.name AS column_name
        ,index_columns.is_descending_key
        ,index_columns.is_included_column
        ,COALESCE(indexes.filter_definition, '') AS predicate
        ,index_columns.key_ordinal
    FROM
        sys.indexes
        JOIN sys.views ON views.object_id = indexes.object_id
        JOIN sys.schemas ON schemas.schema_id = views.schema_id
        JOIN sys.index_columns
            ON index_columns.index_id = indexes.index_id
            AND index_columns.object_id = indexes.object_id
        JOIN sys.columns
            ON columns.column_id = index_columns.column_id
            AND columns.object_id = indexes.object_id
    WHERE
        indexes.name IS NOT NULL
        AND indexes.is_primary_key = 0 AND indexes.is_unique_constraint = 0
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
    ) AS indexed_columns
GROUP BY
    table_schema
    ,table_name
    ,index_name
    ,index_type
    ,is_view_index
    ,is_unique
    ,predicate
ORDER BY
    table_schema
    ,table_name
    ,index_name
;
