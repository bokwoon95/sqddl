CREATE OR REPLACE FUNCTION last_day(TIMESTAMPTZ) RETURNS date AS $$
    SELECT
        CASE
            WHEN EXTRACT(MONTH FROM $1) = 12
            THEN (((EXTRACT(YEAR FROM $1) + 1) || '-01-01')::DATE - INTERVAL '1 day')::DATE
        ELSE
            ((EXTRACT(YEAR FROM $1) || '-' || (EXTRACT(MONTH FROM $1) + 1) || '-01')::DATE - INTERVAL '1 day')::DATE
        END
$$ LANGUAGE sql IMMUTABLE STRICT;
