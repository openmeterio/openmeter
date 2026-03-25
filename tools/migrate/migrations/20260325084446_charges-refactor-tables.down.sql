
DROP VIEW IF EXISTS "charges_search_v1s";


-- reverse: create index "charges_charge_usage_based_id_key" to table: "charges"
DROP INDEX "charges_charge_usage_based_id_key";
-- reverse: create index "charges_charge_flat_fee_id_key" to table: "charges"
DROP INDEX "charges_charge_flat_fee_id_key";
-- reverse: create index "charges_charge_credit_purchase_id_key" to table: "charges"
DROP INDEX "charges_charge_credit_purchase_id_key";
-- reverse: create index "charge_namespace_unique_reference_id" to table: "charges"
DROP INDEX "charge_namespace_unique_reference_id";
-- reverse: modify "charges" table
ALTER TABLE "charges" DROP CONSTRAINT "charges_charge_usage_based_charge", DROP CONSTRAINT "charges_charge_flat_fees_charge", DROP CONSTRAINT "charges_charge_credit_purchases_charge", DROP COLUMN "charge_usage_based_id", DROP COLUMN "charge_flat_fee_id", DROP COLUMN "charge_credit_purchase_id", ADD COLUMN "advance_after" timestamptz NULL, ADD COLUMN "subscription_phase_id" character(26) NULL, ADD COLUMN "subscription_item_id" character(26) NULL, ADD COLUMN "subscription_id" character(26) NULL, ADD COLUMN "customer_id" character(26) NOT NULL, ADD COLUMN "managed_by" character varying NOT NULL, ADD COLUMN "currency" character varying(3) NOT NULL, ADD COLUMN "status" character varying NOT NULL, ADD COLUMN "full_service_period_to" timestamptz NOT NULL, ADD COLUMN "full_service_period_from" timestamptz NOT NULL, ADD COLUMN "billing_period_to" timestamptz NOT NULL, ADD COLUMN "billing_period_from" timestamptz NOT NULL, ADD COLUMN "service_period_to" timestamptz NOT NULL, ADD COLUMN "service_period_from" timestamptz NOT NULL, ADD COLUMN "description" character varying NULL, ADD COLUMN "name" character varying NOT NULL, ADD COLUMN "updated_at" timestamptz NOT NULL, ADD COLUMN "metadata" jsonb NULL, ADD COLUMN "annotations" jsonb NULL;
-- reverse: drop index "charge_namespace_id" from table: "charges"
CREATE UNIQUE INDEX "charge_namespace_id" ON "charges" ("namespace", "id");
-- reverse: create index "chargeusagebased_namespace_customer_id_unique_reference_id" to table: "charge_usage_based"
DROP INDEX "chargeusagebased_namespace_customer_id_unique_reference_id";
-- reverse: create index "chargeusagebased_annotations" to table: "charge_usage_based"
DROP INDEX "chargeusagebased_annotations";
-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_subscriptions_charges_usage_based", DROP CONSTRAINT "charge_usage_based_subscription_phases_charges_usage_based", DROP CONSTRAINT "charge_usage_based_subscription_items_charges_usage_based", DROP CONSTRAINT "charge_usage_based_customers_charges_usage_based", DROP COLUMN "subscription_phase_id", DROP COLUMN "subscription_item_id", DROP COLUMN "subscription_id", DROP COLUMN "customer_id", DROP COLUMN "status_detailed", DROP COLUMN "description", DROP COLUMN "name", DROP COLUMN "deleted_at", DROP COLUMN "updated_at", DROP COLUMN "created_at", DROP COLUMN "metadata", DROP COLUMN "annotations", DROP COLUMN "advance_after", DROP COLUMN "managed_by", DROP COLUMN "currency", DROP COLUMN "unique_reference_id", DROP COLUMN "full_service_period_to", DROP COLUMN "full_service_period_from", DROP COLUMN "billing_period_to", DROP COLUMN "billing_period_from", DROP COLUMN "service_period_to", DROP COLUMN "service_period_from", ADD CONSTRAINT "charge_usage_based_charges_usage_based" FOREIGN KEY ("id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- reverse: create index "chargeflatfee_namespace_customer_id_unique_reference_id" to table: "charge_flat_fees"
DROP INDEX "chargeflatfee_namespace_customer_id_unique_reference_id";
-- reverse: create index "chargeflatfee_annotations" to table: "charge_flat_fees"
DROP INDEX "chargeflatfee_annotations";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_subscriptions_charges_flat_fee", DROP CONSTRAINT "charge_flat_fees_subscription_phases_charges_flat_fee", DROP CONSTRAINT "charge_flat_fees_subscription_items_charges_flat_fee", DROP CONSTRAINT "charge_flat_fees_customers_charges_flat_fee", DROP COLUMN "subscription_phase_id", DROP COLUMN "subscription_item_id", DROP COLUMN "subscription_id", DROP COLUMN "customer_id", DROP COLUMN "description", DROP COLUMN "name", DROP COLUMN "deleted_at", DROP COLUMN "updated_at", DROP COLUMN "created_at", DROP COLUMN "metadata", DROP COLUMN "annotations", DROP COLUMN "advance_after", DROP COLUMN "managed_by", DROP COLUMN "currency", DROP COLUMN "unique_reference_id", DROP COLUMN "status", DROP COLUMN "full_service_period_to", DROP COLUMN "full_service_period_from", DROP COLUMN "billing_period_to", DROP COLUMN "billing_period_from", DROP COLUMN "service_period_to", DROP COLUMN "service_period_from", ADD CONSTRAINT "charge_flat_fees_charges_flat_fee" FOREIGN KEY ("id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- reverse: create index "chargecreditpurchase_namespace_id" to table: "charge_credit_purchases"
DROP INDEX "chargecreditpurchase_namespace_id";
-- reverse: create index "chargecreditpurchase_namespace_customer_id_unique_reference_id" to table: "charge_credit_purchases"
DROP INDEX "chargecreditpurchase_namespace_customer_id_unique_reference_id";
-- reverse: create index "chargecreditpurchase_annotations" to table: "charge_credit_purchases"
DROP INDEX "chargecreditpurchase_annotations";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP CONSTRAINT "charge_credit_purchases_subscriptions_charges_credit_purchase", DROP CONSTRAINT "charge_credit_purchases_subscription_phases_charges_credit_purc", DROP CONSTRAINT "charge_credit_purchases_subscription_items_charges_credit_purch", DROP CONSTRAINT "charge_credit_purchases_customers_charges_credit_purchase", DROP COLUMN "subscription_phase_id", DROP COLUMN "subscription_item_id", DROP COLUMN "subscription_id", DROP COLUMN "customer_id", DROP COLUMN "description", DROP COLUMN "name", DROP COLUMN "deleted_at", DROP COLUMN "updated_at", DROP COLUMN "created_at", DROP COLUMN "metadata", DROP COLUMN "annotations", DROP COLUMN "advance_after", DROP COLUMN "managed_by", DROP COLUMN "currency", DROP COLUMN "unique_reference_id", DROP COLUMN "status", DROP COLUMN "full_service_period_to", DROP COLUMN "full_service_period_from", DROP COLUMN "billing_period_to", DROP COLUMN "billing_period_from", DROP COLUMN "service_period_to", DROP COLUMN "service_period_from", ADD CONSTRAINT "charge_credit_purchases_charges_credit_purchase" FOREIGN KEY ("id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- reverse: drop index "chargeflatfeepayment_namespace_charge_id" from table: "charge_flat_fee_payments"
CREATE UNIQUE INDEX "chargeflatfeepayment_namespace_charge_id" ON "charge_flat_fee_payments" ("namespace", "charge_id");
-- reverse: drop index "chargecreditpurchaseexternalpayment_namespace_charge_id" from table: "charge_credit_purchase_external_payments"
CREATE UNIQUE INDEX "chargecreditpurchaseexternalpayment_namespace_charge_id" ON "charge_credit_purchase_external_payments" ("namespace", "charge_id");

-- regenerate additional constraints etc that are dropped by the table/key deletions in the up migration
ALTER TABLE "charges" ADD CONSTRAINT "charges_customers_charge_intents" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "charges_subscription_items_charge_intents" FOREIGN KEY ("subscription_item_id") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charges_subscription_phases_charge_intents" FOREIGN KEY ("subscription_phase_id") REFERENCES "subscription_phases" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charges_subscriptions_charge_intents" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
CREATE UNIQUE INDEX "charge_namespace_customer_id_unique_reference_id" ON "charges" ("namespace", "customer_id", "unique_reference_id") WHERE ((unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
CREATE INDEX "charge_annotations" ON "charges" USING gin ("annotations");
