-- reverse: create index "chargeusagebasedoverride_namespace_charge_id" to table: "charge_usage_based_overrides"
DROP INDEX "chargeusagebasedoverride_namespace_charge_id";
-- reverse: create index "chargeusagebasedoverrides_tax_code_id" to table: "charge_usage_based_overrides"
DROP INDEX "chargeusagebasedoverrides_tax_code_id";
-- reverse: create index "charge_usage_based_overrides_charge_id_key" to table: "charge_usage_based_overrides"
DROP INDEX "charge_usage_based_overrides_charge_id_key";
-- reverse: create index "chargeusagebasedoverride_id" to table: "charge_usage_based_overrides"
DROP INDEX "chargeusagebasedoverride_id";
-- reverse: create index "chargeusagebasedoverride_namespace" to table: "charge_usage_based_overrides"
DROP INDEX "chargeusagebasedoverride_namespace";
-- reverse: create "charge_usage_based_overrides" table
DROP TABLE "charge_usage_based_overrides";
-- reverse: create index "chargeflatfeeoverride_namespace_charge_id" to table: "charge_flat_fee_overrides"
DROP INDEX "chargeflatfeeoverride_namespace_charge_id";
-- reverse: create index "chargeflatfeeoverrides_tax_code_id" to table: "charge_flat_fee_overrides"
DROP INDEX "chargeflatfeeoverrides_tax_code_id";
-- reverse: create index "charge_flat_fee_overrides_charge_id_key" to table: "charge_flat_fee_overrides"
DROP INDEX "charge_flat_fee_overrides_charge_id_key";
-- reverse: create index "chargeflatfeeoverride_id" to table: "charge_flat_fee_overrides"
DROP INDEX "chargeflatfeeoverride_id";
-- reverse: create index "chargeflatfeeoverride_namespace" to table: "charge_flat_fee_overrides"
DROP INDEX "chargeflatfeeoverride_namespace";
-- reverse: create "charge_flat_fee_overrides" table
DROP TABLE "charge_flat_fee_overrides";
-- reverse: recreate charges_search_v1s view without base_intent_deleted_at
DROP VIEW IF EXISTS "charges_search_v1s";
-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP COLUMN "intent_deleted_at", DROP COLUMN "override_intent_deleted_at", DROP COLUMN "override_present", ADD COLUMN "override_kind" character varying NULL;
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP COLUMN "intent_deleted_at", DROP COLUMN "override_intent_deleted_at", DROP COLUMN "override_present", ADD COLUMN "override_kind" character varying NULL;
CREATE VIEW "charges_search_v1s" AS
SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", 'credit_purchase' AS "type" FROM "charge_credit_purchases" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", 'flat_fee' AS "type" FROM "charge_flat_fees" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", 'usage_based' AS "type" FROM "charge_usage_based";
