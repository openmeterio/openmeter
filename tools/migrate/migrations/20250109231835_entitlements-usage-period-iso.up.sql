-- Let's update all entitlements and convert usage period to ISO format
UPDATE
    entitlements
SET
    usage_period_interval = CASE
        WHEN usage_period_interval = 'YEAR' THEN 'P1Y'
        WHEN usage_period_interval = 'MONTH' THEN 'P1M'
        WHEN usage_period_interval = 'WEEK' THEN 'P1W'
        WHEN usage_period_interval = 'DAY' THEN 'P1D' -- Should not happen as cases up exhaust all programmatic values
        ELSE usage_period_interval
    END;

-- Let's update grants as well
UPDATE
    grants
SET
    recurrence_period = CASE
        WHEN recurrence_period = 'YEAR' THEN 'P1Y'
        WHEN recurrence_period = 'MONTH' THEN 'P1M'
        WHEN recurrence_period = 'WEEK' THEN 'P1W'
        WHEN recurrence_period = 'DAY' THEN 'P1D' -- Should not happen as cases up exhaust all programmatic values
        ELSE recurrence_period
    END;