-- original: 20250606130010_plan-subscription-billing-cadence.up.sql
-- modify "plans" table - add columns as nullable first
ALTER TABLE "plans" ADD COLUMN "billing_cadence" character varying;
ALTER TABLE "plans" ADD COLUMN "pro_rating_config" jsonb;

-- Update existing plans with billing_cadence from the last phase rate cards
-- Find the shortest (most frequent) billing cadence from rate cards in the last phase
WITH last_phase_billing_cadences AS (
  SELECT
    p.id as plan_id,
    prc.billing_cadence
  FROM plans p
  JOIN plan_phases pp ON p.id = pp.plan_id
  JOIN plan_rate_cards prc ON pp.id = prc.phase_id
  WHERE pp.duration IS NULL  -- last phase
    AND pp.deleted_at IS NULL  -- exclude soft-deleted phases
    AND prc.deleted_at IS NULL  -- exclude soft-deleted rate cards
    AND p.deleted_at IS NULL   -- exclude soft-deleted plans
    AND prc.billing_cadence IS NOT NULL
    AND prc.billing_cadence != ''
),
min_cadences AS (
  SELECT
    plan_id,
    billing_cadence,
    ROW_NUMBER() OVER (PARTITION BY plan_id ORDER BY billing_cadence::interval ASC) as rn
  FROM last_phase_billing_cadences
)
UPDATE plans
SET billing_cadence = mc.billing_cadence
FROM min_cadences mc
WHERE plans.id = mc.plan_id
  AND mc.rn = 1;

-- For plans without rate cards or billing cadence, set default to monthly
-- Note: We set defaults for ALL plans (including soft-deleted) to satisfy NOT NULL constraint
UPDATE plans
SET billing_cadence = 'P1M'
WHERE billing_cadence IS NULL;

-- Set default pro_rating_config for all plans
-- Note: We set defaults for ALL plans (including soft-deleted) to satisfy NOT NULL constraint
UPDATE plans
SET pro_rating_config = '{"mode": "prorate_prices", "enabled": true}'
WHERE pro_rating_config IS NULL;

-- Now make the columns required
ALTER TABLE "plans" ALTER COLUMN "billing_cadence" SET NOT NULL;
ALTER TABLE "plans" ALTER COLUMN "pro_rating_config" SET NOT NULL;

-- modify "subscriptions" table - add columns as nullable first
ALTER TABLE "subscriptions" ADD COLUMN "billing_cadence" character varying;
ALTER TABLE "subscriptions" ADD COLUMN "pro_rating_config" jsonb;

-- Update existing subscriptions with billing_cadence from the last phase subscription items
-- Find the shortest (most frequent) billing cadence from subscription items in the last phase
WITH subscription_phase_order_ranks AS (
  SELECT
    sp.id,
    sp.subscription_id,
    ROW_NUMBER() OVER (PARTITION BY sp.subscription_id ORDER BY sp.active_from DESC) as rn
  FROM subscription_phases sp
  JOIN subscriptions s ON sp.subscription_id = s.id
  WHERE sp.deleted_at IS NULL  -- exclude soft-deleted phases
    AND s.deleted_at IS NULL   -- exclude soft-deleted subscriptions
), last_phase_billing_cadences AS (
  SELECT
    spor.subscription_id,
    si.billing_cadence
  FROM subscription_phase_order_ranks spor
  JOIN subscription_items si ON spor.id = si.phase_id
  WHERE spor.rn = 1 -- last phase
    AND si.deleted_at IS NULL  -- exclude soft-deleted subscription items
    AND si.billing_cadence IS NOT NULL
    AND si.billing_cadence != ''
), min_cadences AS (
  SELECT
    subscription_id,
    billing_cadence,
    ROW_NUMBER() OVER (PARTITION BY subscription_id ORDER BY billing_cadence::interval ASC) as rn
  FROM last_phase_billing_cadences
)
UPDATE subscriptions
SET billing_cadence = mc.billing_cadence
FROM min_cadences mc
WHERE subscriptions.id = mc.subscription_id
  AND mc.rn = 1;

-- For subscriptions without rate cards or billing cadence, set default to monthly
-- Note: We set defaults for ALL subscriptions (including soft-deleted) to satisfy NOT NULL constraint
UPDATE subscriptions
SET billing_cadence = 'P1M'
WHERE billing_cadence IS NULL;

-- Set default pro_rating_config for all subscriptions
-- Note: We set defaults for ALL subscriptions (including soft-deleted) to satisfy NOT NULL constraint
UPDATE subscriptions
SET pro_rating_config = '{"mode": "prorate_prices", "enabled": true}'
WHERE pro_rating_config IS NULL;

-- Now make the columns required
ALTER TABLE "subscriptions" ALTER COLUMN "billing_cadence" SET NOT NULL;
ALTER TABLE "subscriptions" ALTER COLUMN "pro_rating_config" SET NOT NULL;
