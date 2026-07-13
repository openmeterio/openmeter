-- drop dependent search view before widening charge currency columns
DROP VIEW IF EXISTS "charges_search_v1s";
-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ALTER COLUMN "currency" TYPE character varying(24);
-- modify "charge_flat_fee_run_detailed_lines" table
ALTER TABLE "charge_flat_fee_run_detailed_lines" ALTER COLUMN "currency" TYPE character varying(24);
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ALTER COLUMN "currency" TYPE character varying(24);
-- modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ALTER COLUMN "currency" TYPE character varying(24);
-- modify "charge_usage_based_run_detailed_line" table
ALTER TABLE "charge_usage_based_run_detailed_line" ALTER COLUMN "currency" TYPE character varying(24);
-- modify "credit_realization_lineages" table
ALTER TABLE "credit_realization_lineages" ALTER COLUMN "currency" TYPE character varying(24);
-- modify "ledger_breakage_records" table
ALTER TABLE "ledger_breakage_records" ALTER COLUMN "currency" TYPE character varying(24);
-- modify "ledger_sub_account_routes" table
ALTER TABLE "ledger_sub_account_routes" ADD COLUMN "source" character varying NULL;
-- recreate charges_search_v1s view after widening charge currency columns
CREATE VIEW "charges_search_v1s" AS
SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", NULL::timestamptz AS "base_intent_deleted_at", 'credit_purchase' AS "type" FROM "charge_credit_purchases" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", "intent_deleted_at" AS "base_intent_deleted_at", 'flat_fee' AS "type" FROM "charge_flat_fees" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", "intent_deleted_at" AS "base_intent_deleted_at", 'usage_based' AS "type" FROM "charge_usage_based";
