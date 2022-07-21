SELECT
    extname AS extension_name
    ,extversion AS extension_version
FROM
    pg_extension
WHERE
    TRUE
    {{- if .Extensions }}
    AND extname IN ({{ mklist .Extensions }})
    {{- else if .ExcludeExtensions }}
    AND extname NOT IN ({{ mklist .ExcludeExtensions }})
    {{- end }}
ORDER BY
    extname
;
