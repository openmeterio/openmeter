-- Check compatibility before dropping attribution needed by plan-aware filters.
BEGIN;
DO $migration$
BEGIN
  IF EXISTS (SELECT 1 FROM "charge_credit_purchases" WHERE "filters"->'schema_version' IS DISTINCT FROM '1'::jsonb OR "filters" - 'schema_version' - 'features' <> '{}'::jsonb)
     OR EXISTS (SELECT 1 FROM "ledger_sub_account_routes" WHERE "filters"->'schema_version' IS DISTINCT FROM '1'::jsonb OR "filters" - 'schema_version' - 'features' <> '{}'::jsonb) THEN
    RAISE EXCEPTION 'cannot remove plan attribution with unsupported credit filter versions or dimensions';
  END IF;
END
$migration$;

-- Retain the appended view column for dependent readers during rollback.
CREATE OR REPLACE VIEW "charges_search_v1s" AS
SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "custom_currency_id", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", NULL::timestamptz AS "base_intent_deleted_at", 'credit_purchase' AS "type", NULL::char(26) AS "feature_id", NULL::varchar AS "feature_key", NULL::jsonb AS "subscription_plan" FROM "charge_credit_purchases" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "custom_currency_id", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", "intent_deleted_at" AS "base_intent_deleted_at", 'flat_fee' AS "type", "feature_id" AS "feature_id", "feature_key" AS "feature_key", NULL::jsonb AS "subscription_plan" FROM "charge_flat_fees" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "custom_currency_id", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", "intent_deleted_at" AS "base_intent_deleted_at", 'usage_based' AS "type", "feature_id" AS "feature_id", "feature_key" AS "feature_key", NULL::jsonb AS "subscription_plan" FROM "charge_usage_based";

-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP COLUMN "subscription_plan";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP COLUMN "subscription_plan";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP COLUMN "subscription_plan";

COMMIT;
