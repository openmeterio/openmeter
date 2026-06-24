-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP COLUMN "override_kind", ADD COLUMN "override_present" boolean NOT NULL DEFAULT false, ADD COLUMN "override_intent_deleted_at" timestamptz NULL, ADD COLUMN "intent_deleted_at" timestamptz NULL;
-- modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP COLUMN "override_kind", ADD COLUMN "override_present" boolean NOT NULL DEFAULT false, ADD COLUMN "override_intent_deleted_at" timestamptz NULL, ADD COLUMN "intent_deleted_at" timestamptz NULL;
-- create "charge_flat_fee_overrides" table
CREATE TABLE "charge_flat_fee_overrides" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "metadata" jsonb NULL,
  "tax_behavior" character varying NULL,
  "tax_code_id" character(26) NULL,
  "intent_deleted_at" timestamptz NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "full_service_period_from" timestamptz NOT NULL,
  "full_service_period_to" timestamptz NOT NULL,
  "billing_period_from" timestamptz NOT NULL,
  "billing_period_to" timestamptz NOT NULL,
  "feature_key" character varying NULL,
  "payment_term" character varying NOT NULL,
  "pro_rating" jsonb NOT NULL,
  "amount_before_proration" numeric NOT NULL,
  "percentage_discounts" jsonb NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_flat_fee_overrides_charge_flat_fees_intent_override" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeflatfeeoverride_namespace" to table: "charge_flat_fee_overrides"
CREATE INDEX "chargeflatfeeoverride_namespace" ON "charge_flat_fee_overrides" ("namespace");
-- create index "chargeflatfeeoverride_id" to table: "charge_flat_fee_overrides"
CREATE UNIQUE INDEX "chargeflatfeeoverride_id" ON "charge_flat_fee_overrides" ("id");
-- create index "charge_flat_fee_overrides_charge_id_key" to table: "charge_flat_fee_overrides"
CREATE UNIQUE INDEX "charge_flat_fee_overrides_charge_id_key" ON "charge_flat_fee_overrides" ("charge_id");
-- create index "chargeflatfeeoverride_namespace_charge_id" to table: "charge_flat_fee_overrides"
CREATE UNIQUE INDEX "chargeflatfeeoverride_namespace_charge_id" ON "charge_flat_fee_overrides" ("namespace", "charge_id");
-- create "charge_usage_based_overrides" table
CREATE TABLE "charge_usage_based_overrides" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "metadata" jsonb NULL,
  "tax_behavior" character varying NULL,
  "tax_code_id" character(26) NULL,
  "intent_deleted_at" timestamptz NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "full_service_period_from" timestamptz NOT NULL,
  "full_service_period_to" timestamptz NOT NULL,
  "billing_period_from" timestamptz NOT NULL,
  "billing_period_to" timestamptz NOT NULL,
  "feature_key" character varying NOT NULL,
  "price" jsonb NOT NULL,
  "discounts" jsonb NOT NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_usage_based_overrides_charge_usage_based_intent_override" FOREIGN KEY ("charge_id") REFERENCES "charge_usage_based" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeusagebasedoverride_namespace" to table: "charge_usage_based_overrides"
CREATE INDEX "chargeusagebasedoverride_namespace" ON "charge_usage_based_overrides" ("namespace");
-- create index "chargeusagebasedoverride_id" to table: "charge_usage_based_overrides"
CREATE UNIQUE INDEX "chargeusagebasedoverride_id" ON "charge_usage_based_overrides" ("id");
-- create index "charge_usage_based_overrides_charge_id_key" to table: "charge_usage_based_overrides"
CREATE UNIQUE INDEX "charge_usage_based_overrides_charge_id_key" ON "charge_usage_based_overrides" ("charge_id");
-- create index "chargeusagebasedoverride_namespace_charge_id" to table: "charge_usage_based_overrides"
CREATE UNIQUE INDEX "chargeusagebasedoverride_namespace_charge_id" ON "charge_usage_based_overrides" ("namespace", "charge_id");
-- recreate charges_search_v1s view to expose concrete intent_deleted_at as base_intent_deleted_at
DROP VIEW IF EXISTS "charges_search_v1s";
CREATE VIEW "charges_search_v1s" AS
SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", NULL::timestamptz AS "base_intent_deleted_at", 'credit_purchase' AS "type" FROM "charge_credit_purchases" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", "intent_deleted_at" AS "base_intent_deleted_at", 'flat_fee' AS "type" FROM "charge_flat_fees" UNION ALL SELECT "id", "namespace", "metadata", "created_at", "updated_at", "deleted_at", "name", "description", "annotations", "customer_id", "service_period_from", "service_period_to", "billing_period_from", "billing_period_to", "full_service_period_from", "full_service_period_to", "status", "unique_reference_id", "currency", "managed_by", "subscription_id", "subscription_phase_id", "subscription_item_id", "advance_after", "tax_code_id", "tax_behavior", "intent_deleted_at" AS "base_intent_deleted_at", 'usage_based' AS "type" FROM "charge_usage_based";
