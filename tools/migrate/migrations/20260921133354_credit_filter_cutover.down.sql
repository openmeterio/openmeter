-- Restore feature projections before rolling back the application.
BEGIN;

DO $migration$
BEGIN
  IF EXISTS (SELECT 1 FROM "charge_credit_purchases" WHERE "filters"->'schema_version' IS DISTINCT FROM '1'::jsonb OR "filters" - 'schema_version' - 'features' <> '{}'::jsonb) THEN
    RAISE EXCEPTION 'cannot restore feature-only storage with unsupported credit filter versions or dimensions';
  END IF;
END
$migration$;
UPDATE "charge_credit_purchases"
SET "feature_filters" = NULLIF(ARRAY(SELECT jsonb_array_elements_text("filters"->'features')), ARRAY[]::text[]);

DO $migration$
BEGIN
  IF EXISTS (SELECT 1 FROM "ledger_sub_account_routes" WHERE "filters"->'schema_version' IS DISTINCT FROM '1'::jsonb OR "filters" - 'schema_version' - 'features' <> '{}'::jsonb) THEN
    RAISE EXCEPTION 'cannot restore feature-only storage with unsupported credit filter versions or dimensions';
  END IF;
END
$migration$;
UPDATE "ledger_sub_account_routes"
SET "features" = NULLIF(ARRAY(SELECT jsonb_array_elements_text("filters"->'features')), ARRAY[]::text[]);

-- reverse: modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" ALTER COLUMN "filters" DROP NOT NULL;
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ALTER COLUMN "filters" DROP NOT NULL;

COMMIT;
