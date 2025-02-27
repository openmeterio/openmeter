-- modify "usage_resets" table
ALTER TABLE
    "usage_resets"
ADD
    COLUMN "anchor" timestamptz NOT NULL DEFAULT '2025-02-01 23:18:35';

-- Let's drop the default used for creation
ALTER TABLE
    "usage_resets"
ALTER COLUMN
    "anchor" DROP DEFAULT;

-- we need to update all existing the LAST reset for each entitlement so anchor has the same value as entitlement.usage_period_anchor
-- As the anchor field is not nullable, we default it to the reset time.
-- This WORKS because entitlements could only be reset manually before this:
-- 1. between each two subsequent reset it is impossible to have a full usage period
-- 2. setting the anchor time to the reset time guarantees that said period starts with the reset time
-- therefore the automatic reset calculation wont find any "ghost" periods inbetween the historical resets
WITH latest_resets AS (
    SELECT
        id,
        entitlement_id,
        reset_time,
        ROW_NUMBER() OVER (
            PARTITION BY entitlement_id
            ORDER BY
                reset_time DESC
        ) as rn
    FROM
        usage_resets
)
UPDATE
    usage_resets
SET
    anchor = CASE
        WHEN lr.rn = 1 THEN e.usage_period_anchor
        ELSE lr.reset_time
    END
FROM
    entitlements e
    JOIN latest_resets lr ON lr.entitlement_id = e.id
WHERE
    usage_resets.id = lr.id;