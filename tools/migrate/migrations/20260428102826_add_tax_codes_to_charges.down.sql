-- reverse: recreate charges_search_v1s view without tax_code_id and tax_behavior columns
DROP VIEW IF EXISTS "charges_search_v1s";
CREATE VIEW "charges_search_v1s" AS
SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", 'credit_purchase' AS "type" FROM "charge_credit_purchases" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", 'flat_fee' AS "type" FROM "charge_flat_fees" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", 'usage_based' AS "type" FROM "charge_usage_based";
-- reverse: drop indexes on tax_code_id
DROP INDEX IF EXISTS "chargeusagebased_tax_code_id";
DROP INDEX IF EXISTS "chargeflatfees_tax_code_id";
DROP INDEX IF EXISTS "chargecreditpurchases_tax_code_id";
-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_tax_codes_charge_usage_based", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_tax_codes_charge_flat_fees", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchases_tax_codes_charge_credit_purchases", DROP COLUMN "tax_code_id", DROP COLUMN "tax_behavior";
