-- In all honesty we don't have much of a chance here in retrofitting the values. What we can do is keep them as is and use the deleted_at flag so the system will ignore them going on.
UPDATE
    entitlements
SET
    deleted_at = NOW()
WHERE
    created_at > '2025-01-09 23:18:35' :: timestamp -- time of the migration file
    AND usage_period_interval NOT IN ('P1Y', 'P1M', 'P1W', 'P1D');

UPDATE
    entitlements
SET
    usage_period_interval = CASE
        WHEN usage_period_interval = 'P1Y' THEN 'YEAR'
        WHEN usage_period_interval = 'P1M' THEN 'MONTH'
        WHEN usage_period_interval = 'P1W' THEN 'WEEK'
        WHEN usage_period_interval = 'P1D' THEN 'DAY'
        ELSE usage_period_interval
    END;

UPDATE
    grants
SET
    deleted_at = NOW()
WHERE
    created_at > '2025-01-09 23:18:35' :: timestamp
    AND recurrence_period NOT IN ('P1Y', 'P1M', 'P1W', 'P1D');

UPDATE
    grants
SET
    recurrence_period = CASE
        WHEN recurrence_period = 'P1Y' THEN 'YEAR'
        WHEN recurrence_period = 'P1M' THEN 'MONTH'
        WHEN recurrence_period = 'P1W' THEN 'WEEK'
        WHEN recurrence_period = 'P1D' THEN 'DAY'
        ELSE recurrence_period
    END;