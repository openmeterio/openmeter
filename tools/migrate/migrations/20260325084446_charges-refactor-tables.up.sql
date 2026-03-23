-- drop index "chargecreditpurchaseexternalpayment_namespace_charge_id" from table: "charge_credit_purchase_external_payments"
DROP INDEX "chargecreditpurchaseexternalpayment_namespace_charge_id";
-- drop index "chargeflatfeepayment_namespace_charge_id" from table: "charge_flat_fee_payments"
DROP INDEX "chargeflatfeepayment_namespace_charge_id";
-- modify "charge_credit_purchases" table
-- atlas:nolint MF103 CD101
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchases_charges_credit_purchase", ADD COLUMN "service_period_from" timestamptz NOT NULL, ADD COLUMN "service_period_to" timestamptz NOT NULL, ADD COLUMN "billing_period_from" timestamptz NOT NULL, ADD COLUMN "billing_period_to" timestamptz NOT NULL, ADD COLUMN "full_service_period_from" timestamptz NOT NULL, ADD COLUMN "full_service_period_to" timestamptz NOT NULL, ADD COLUMN "status" character varying NOT NULL, ADD COLUMN "unique_reference_id" character varying NULL, ADD COLUMN "currency" character varying(3) NOT NULL, ADD COLUMN "managed_by" character varying NOT NULL, ADD COLUMN "advance_after" timestamptz NULL, ADD COLUMN "annotations" jsonb NULL, ADD COLUMN "metadata" jsonb NULL, ADD COLUMN "created_at" timestamptz NOT NULL, ADD COLUMN "updated_at" timestamptz NOT NULL, ADD COLUMN "deleted_at" timestamptz NULL, ADD COLUMN "name" character varying NOT NULL, ADD COLUMN "description" character varying NULL, ADD COLUMN "customer_id" character(26) NOT NULL, ADD COLUMN "subscription_id" character(26) NULL, ADD COLUMN "subscription_item_id" character(26) NULL, ADD COLUMN "subscription_phase_id" character(26) NULL, ADD CONSTRAINT "charge_credit_purchases_customers_charges_credit_purchase" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "charge_credit_purchases_subscription_items_charges_credit_purch" FOREIGN KEY ("subscription_item_id") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_credit_purchases_subscription_phases_charges_credit_purc" FOREIGN KEY ("subscription_phase_id") REFERENCES "subscription_phases" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_credit_purchases_subscriptions_charges_credit_purchase" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "chargecreditpurchase_annotations" to table: "charge_credit_purchases"
CREATE INDEX "chargecreditpurchase_annotations" ON "charge_credit_purchases" USING gin ("annotations");
-- create index "chargecreditpurchase_namespace_customer_id_unique_reference_id" to table: "charge_credit_purchases"
CREATE UNIQUE INDEX "chargecreditpurchase_namespace_customer_id_unique_reference_id" ON "charge_credit_purchases" ("namespace", "customer_id", "unique_reference_id") WHERE ((unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- create index "chargecreditpurchase_namespace_id" to table: "charge_credit_purchases"
CREATE UNIQUE INDEX "chargecreditpurchase_namespace_id" ON "charge_credit_purchases" ("namespace", "id");
-- modify "charge_flat_fees" table
-- atlas:nolint CD101 MF103
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_charges_flat_fee", ADD COLUMN "service_period_from" timestamptz NOT NULL, ADD COLUMN "service_period_to" timestamptz NOT NULL, ADD COLUMN "billing_period_from" timestamptz NOT NULL, ADD COLUMN "billing_period_to" timestamptz NOT NULL, ADD COLUMN "full_service_period_from" timestamptz NOT NULL, ADD COLUMN "full_service_period_to" timestamptz NOT NULL, ADD COLUMN "status" character varying NOT NULL, ADD COLUMN "unique_reference_id" character varying NULL, ADD COLUMN "currency" character varying(3) NOT NULL, ADD COLUMN "managed_by" character varying NOT NULL, ADD COLUMN "advance_after" timestamptz NULL, ADD COLUMN "annotations" jsonb NULL, ADD COLUMN "metadata" jsonb NULL, ADD COLUMN "created_at" timestamptz NOT NULL, ADD COLUMN "updated_at" timestamptz NOT NULL, ADD COLUMN "deleted_at" timestamptz NULL, ADD COLUMN "name" character varying NOT NULL, ADD COLUMN "description" character varying NULL, ADD COLUMN "customer_id" character(26) NOT NULL, ADD COLUMN "subscription_id" character(26) NULL, ADD COLUMN "subscription_item_id" character(26) NULL, ADD COLUMN "subscription_phase_id" character(26) NULL, ADD CONSTRAINT "charge_flat_fees_customers_charges_flat_fee" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "charge_flat_fees_subscription_items_charges_flat_fee" FOREIGN KEY ("subscription_item_id") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_flat_fees_subscription_phases_charges_flat_fee" FOREIGN KEY ("subscription_phase_id") REFERENCES "subscription_phases" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_flat_fees_subscriptions_charges_flat_fee" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "chargeflatfee_annotations" to table: "charge_flat_fees"
CREATE INDEX "chargeflatfee_annotations" ON "charge_flat_fees" USING gin ("annotations");
-- create index "chargeflatfee_namespace_customer_id_unique_reference_id" to table: "charge_flat_fees"
CREATE UNIQUE INDEX "chargeflatfee_namespace_customer_id_unique_reference_id" ON "charge_flat_fees" ("namespace", "customer_id", "unique_reference_id") WHERE ((unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- modify "charge_usage_based" table
-- atlas:nolint CD101 MF103
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_charges_usage_based", ADD COLUMN "service_period_from" timestamptz NOT NULL, ADD COLUMN "service_period_to" timestamptz NOT NULL, ADD COLUMN "billing_period_from" timestamptz NOT NULL, ADD COLUMN "billing_period_to" timestamptz NOT NULL, ADD COLUMN "full_service_period_from" timestamptz NOT NULL, ADD COLUMN "full_service_period_to" timestamptz NOT NULL, ADD COLUMN "unique_reference_id" character varying NULL, ADD COLUMN "currency" character varying(3) NOT NULL, ADD COLUMN "managed_by" character varying NOT NULL, ADD COLUMN "advance_after" timestamptz NULL, ADD COLUMN "annotations" jsonb NULL, ADD COLUMN "metadata" jsonb NULL, ADD COLUMN "created_at" timestamptz NOT NULL, ADD COLUMN "updated_at" timestamptz NOT NULL, ADD COLUMN "deleted_at" timestamptz NULL, ADD COLUMN "name" character varying NOT NULL, ADD COLUMN "description" character varying NULL, ADD COLUMN "status_detailed" character varying NOT NULL, ADD COLUMN "customer_id" character(26) NOT NULL, ADD COLUMN "subscription_id" character(26) NULL, ADD COLUMN "subscription_item_id" character(26) NULL, ADD COLUMN "subscription_phase_id" character(26) NULL, ADD CONSTRAINT "charge_usage_based_customers_charges_usage_based" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "charge_usage_based_subscription_items_charges_usage_based" FOREIGN KEY ("subscription_item_id") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_usage_based_subscription_phases_charges_usage_based" FOREIGN KEY ("subscription_phase_id") REFERENCES "subscription_phases" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_usage_based_subscriptions_charges_usage_based" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "chargeusagebased_annotations" to table: "charge_usage_based"
CREATE INDEX "chargeusagebased_annotations" ON "charge_usage_based" USING gin ("annotations");
-- create index "chargeusagebased_namespace_customer_id_unique_reference_id" to table: "charge_usage_based"
CREATE UNIQUE INDEX "chargeusagebased_namespace_customer_id_unique_reference_id" ON "charge_usage_based" ("namespace", "customer_id", "unique_reference_id") WHERE ((unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- drop index "charge_namespace_id" from table: "charges"
DROP INDEX "charge_namespace_id";
-- modify "charges" table
-- atlas:nolint DS103
ALTER TABLE "charges" DROP COLUMN "annotations", DROP COLUMN "metadata", DROP COLUMN "updated_at", DROP COLUMN "name", DROP COLUMN "description", DROP COLUMN "service_period_from", DROP COLUMN "service_period_to", DROP COLUMN "billing_period_from", DROP COLUMN "billing_period_to", DROP COLUMN "full_service_period_from", DROP COLUMN "full_service_period_to", DROP COLUMN "status", DROP COLUMN "currency", DROP COLUMN "managed_by", DROP COLUMN "customer_id", DROP COLUMN "subscription_id", DROP COLUMN "subscription_item_id", DROP COLUMN "subscription_phase_id", DROP COLUMN "advance_after", ADD COLUMN "charge_credit_purchase_id" character(26) NULL, ADD COLUMN "charge_flat_fee_id" character(26) NULL, ADD COLUMN "charge_usage_based_id" character(26) NULL, ADD CONSTRAINT "charges_charge_credit_purchases_charge" FOREIGN KEY ("charge_credit_purchase_id") REFERENCES "charge_credit_purchases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "charges_charge_flat_fees_charge" FOREIGN KEY ("charge_flat_fee_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "charges_charge_usage_based_charge" FOREIGN KEY ("charge_usage_based_id") REFERENCES "charge_usage_based" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- create index "charge_namespace_unique_reference_id" to table: "charges"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "charge_namespace_unique_reference_id" ON "charges" ("namespace", "unique_reference_id") WHERE ((unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- create index "charges_charge_credit_purchase_id_key" to table: "charges"
CREATE UNIQUE INDEX "charges_charge_credit_purchase_id_key" ON "charges" ("charge_credit_purchase_id");
-- create index "charges_charge_flat_fee_id_key" to table: "charges"
CREATE UNIQUE INDEX "charges_charge_flat_fee_id_key" ON "charges" ("charge_flat_fee_id");
-- create index "charges_charge_usage_based_id_key" to table: "charges"
CREATE UNIQUE INDEX "charges_charge_usage_based_id_key" ON "charges" ("charge_usage_based_id");


CREATE VIEW "charges_search_v1s" AS
SELECT
  "id",
  "namespace",
  "metadata",
  "created_at",
  "updated_at",
  "deleted_at",
  "name",
  "description",
  "annotations",
  "customer_id",
  "service_period_from",
  "service_period_to",
  "billing_period_from",
  "billing_period_to",
  "full_service_period_from",
  "full_service_period_to",
  "status",
  "unique_reference_id",
  "currency",
  "managed_by",
  "subscription_id",
  "subscription_phase_id",
  "subscription_item_id",
  "advance_after",
  'credit_purchase' AS "type"
FROM "charge_credit_purchases"
UNION ALL
SELECT
  "id",
  "namespace",
  "metadata",
  "created_at",
  "updated_at",
  "deleted_at",
  "name",
  "description",
  "annotations",
  "customer_id",
  "service_period_from",
  "service_period_to",
  "billing_period_from",
  "billing_period_to",
  "full_service_period_from",
  "full_service_period_to",
  "status",
  "unique_reference_id",
  "currency",
  "managed_by",
  "subscription_id",
  "subscription_phase_id",
  "subscription_item_id",
  "advance_after",
  'flat_fee' AS "type"
FROM "charge_flat_fees"
UNION ALL
SELECT
  "id",
  "namespace",
  "metadata",
  "created_at",
  "updated_at",
  "deleted_at",
  "name",
  "description",
  "annotations",
  "customer_id",
  "service_period_from",
  "service_period_to",
  "billing_period_from",
  "billing_period_to",
  "full_service_period_from",
  "full_service_period_to",
  "status",
  "unique_reference_id",
  "currency",
  "managed_by",
  "subscription_id",
  "subscription_phase_id",
  "subscription_item_id",
  "advance_after",
  'usage_based' AS "type"
FROM "charge_usage_based";
