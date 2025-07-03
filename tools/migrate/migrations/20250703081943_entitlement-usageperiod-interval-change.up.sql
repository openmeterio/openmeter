-- modify "usage_resets" table
ALTER TABLE "usage_resets" ADD COLUMN "usage_period_interval" character varying NULL;

-- Update all existing usage_resets with the usage_period_interval from their corresponding entitlement
UPDATE usage_resets
SET usage_period_interval = e.usage_period_interval
FROM entitlements e
WHERE usage_resets.entitlement_id = e.id;

-- Now make the column NOT NULL
-- atlas:nolint MF104
ALTER TABLE "usage_resets" ALTER COLUMN "usage_period_interval" SET NOT NULL;
