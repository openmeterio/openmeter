-- Deploy the dual-writing storage release everywhere before applying this migration.
BEGIN;

UPDATE "charge_credit_purchases"
SET "filters" = jsonb_strip_nulls(jsonb_build_object(
  'schema_version', 1,
  'features', NULLIF(to_jsonb("feature_filters"), '[]'::jsonb)
))
WHERE "filters" IS NULL;

UPDATE "ledger_sub_account_routes"
SET "filters" = jsonb_strip_nulls(jsonb_build_object(
  'schema_version', 1,
  'features', NULLIF(to_jsonb("features"), '[]'::jsonb)
))
WHERE "filters" IS NULL;

-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ALTER COLUMN "filters" SET NOT NULL;
-- modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" ALTER COLUMN "filters" SET NOT NULL;

COMMIT;
