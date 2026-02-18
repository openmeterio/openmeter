-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "billing_invoice_line_standard_invoice_settlments" character(26) NULL, ADD COLUMN "charge_id" character(26) NULL;
-- modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" ADD COLUMN "charge_id" character(26) NULL;
-- create "charge_credit_purchases" table
CREATE TABLE "charge_credit_purchases" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "currency" character varying(3) NOT NULL,
  "credit_amount" numeric NOT NULL,
  "settlement" jsonb NOT NULL,
  "status" character varying NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "chargecreditpurchase_id" to table: "charge_credit_purchases"
CREATE UNIQUE INDEX "chargecreditpurchase_id" ON "charge_credit_purchases" ("id");
-- create index "chargecreditpurchase_namespace" to table: "charge_credit_purchases"
CREATE INDEX "chargecreditpurchase_namespace" ON "charge_credit_purchases" ("namespace");
-- create "charge_flat_fees" table
CREATE TABLE "charge_flat_fees" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "payment_term" character varying NOT NULL,
  "invoice_at" timestamptz NOT NULL,
  "settlement_mode" character varying NOT NULL,
  "discounts" jsonb NULL,
  "pro_rating" character varying NOT NULL,
  "feature_key" character varying NULL,
  "amount_before_proration" numeric NOT NULL,
  "amount_after_proration" numeric NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "chargeflatfee_id" to table: "charge_flat_fees"
CREATE UNIQUE INDEX "chargeflatfee_id" ON "charge_flat_fees" ("id");
-- create index "chargeflatfee_namespace" to table: "charge_flat_fees"
CREATE INDEX "chargeflatfee_namespace" ON "charge_flat_fees" ("namespace");
-- create index "chargeflatfee_namespace_id" to table: "charge_flat_fees"
CREATE UNIQUE INDEX "chargeflatfee_namespace_id" ON "charge_flat_fees" ("namespace", "id");
-- create "charge_usage_baseds" table
CREATE TABLE "charge_usage_baseds" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "price" jsonb NOT NULL,
  "feature_key" character varying NOT NULL,
  "invoice_at" timestamptz NOT NULL,
  "tax_config" jsonb NULL,
  "settlement_mode" character varying NOT NULL,
  "discounts" jsonb NULL,
  PRIMARY KEY ("id")
);
-- create index "chargeusagebased_id" to table: "charge_usage_baseds"
CREATE UNIQUE INDEX "chargeusagebased_id" ON "charge_usage_baseds" ("id");
-- create index "chargeusagebased_namespace" to table: "charge_usage_baseds"
CREATE INDEX "chargeusagebased_namespace" ON "charge_usage_baseds" ("namespace");
-- create index "chargeusagebased_namespace_id" to table: "charge_usage_baseds"
CREATE UNIQUE INDEX "chargeusagebased_namespace_id" ON "charge_usage_baseds" ("namespace", "id");
-- create "charges" table
CREATE TABLE "charges" (
  "id" character(26) NOT NULL,
  "annotations" jsonb NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "billing_period_from" timestamptz NOT NULL,
  "billing_period_to" timestamptz NOT NULL,
  "full_service_period_from" timestamptz NOT NULL,
  "full_service_period_to" timestamptz NOT NULL,
  "type" character varying NOT NULL,
  "status" character varying NOT NULL,
  "unique_reference_id" character varying NULL,
  "currency" character varying(3) NOT NULL,
  "managed_by" character varying NOT NULL,
  "customer_id" character(26) NOT NULL,
  "subscription_id" character(26) NULL,
  "subscription_item_id" character(26) NULL,
  "subscription_phase_id" character(26) NULL,
  PRIMARY KEY ("id")
);
-- create index "charge_annotations" to table: "charges"
CREATE INDEX "charge_annotations" ON "charges" USING gin ("annotations");
-- create index "charge_id" to table: "charges"
CREATE UNIQUE INDEX "charge_id" ON "charges" ("id");
-- create index "charge_namespace" to table: "charges"
CREATE INDEX "charge_namespace" ON "charges" ("namespace");
-- create index "charge_namespace_customer_id_unique_reference_id" to table: "charges"
CREATE UNIQUE INDEX "charge_namespace_customer_id_unique_reference_id" ON "charges" ("namespace", "customer_id", "unique_reference_id") WHERE ((unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- create index "charge_namespace_id" to table: "charges"
CREATE UNIQUE INDEX "charge_namespace_id" ON "charges" ("namespace", "id");
-- create "standard_invoice_settlements" table
CREATE TABLE "standard_invoice_settlements" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "amount" numeric NOT NULL,
  "taxes_total" numeric NOT NULL,
  "taxes_inclusive_total" numeric NOT NULL,
  "taxes_exclusive_total" numeric NOT NULL,
  "charges_total" numeric NOT NULL,
  "discounts_total" numeric NOT NULL,
  "total" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "metered_service_period_quantity" numeric NOT NULL,
  "metered_pre_service_period_quantity" numeric NOT NULL,
  "charge_id" character(26) NOT NULL,
  "line_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "standardinvoicesettlement_annotations" to table: "standard_invoice_settlements"
CREATE INDEX "standardinvoicesettlement_annotations" ON "standard_invoice_settlements" USING gin ("annotations");
-- create index "standardinvoicesettlement_id" to table: "standard_invoice_settlements"
CREATE UNIQUE INDEX "standardinvoicesettlement_id" ON "standard_invoice_settlements" ("id");
-- create index "standardinvoicesettlement_namespace" to table: "standard_invoice_settlements"
CREATE INDEX "standardinvoicesettlement_namespace" ON "standard_invoice_settlements" ("namespace");
-- create index "standardinvoicesettlement_namespace_charge_id_line_id" to table: "standard_invoice_settlements"
CREATE UNIQUE INDEX "standardinvoicesettlement_namespace_charge_id_line_id" ON "standard_invoice_settlements" ("namespace", "charge_id", "line_id");
-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD CONSTRAINT "billing_invoice_lines_charges_billing_invoice_lines" FOREIGN KEY ("charge_id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "billing_invoice_lines_standard_invoice_settlements_standard_inv" FOREIGN KEY ("billing_invoice_line_standard_invoice_settlments") REFERENCES "standard_invoice_settlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "billing_invoice_split_line_groups" table
ALTER TABLE "billing_invoice_split_line_groups" ADD CONSTRAINT "billing_invoice_split_line_groups_charges_billing_split_line_gr" FOREIGN KEY ("charge_id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD CONSTRAINT "charge_credit_purchases_charges_credit_purchase" FOREIGN KEY ("id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD CONSTRAINT "charge_flat_fees_charges_flat_fee" FOREIGN KEY ("id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- modify "charge_usage_baseds" table
ALTER TABLE "charge_usage_baseds" ADD CONSTRAINT "charge_usage_baseds_charges_usage_based" FOREIGN KEY ("id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- modify "charges" table
ALTER TABLE "charges" ADD CONSTRAINT "charges_customers_charge_intents" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "charges_subscription_items_charge_intents" FOREIGN KEY ("subscription_item_id") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charges_subscription_phases_charge_intents" FOREIGN KEY ("subscription_phase_id") REFERENCES "subscription_phases" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charges_subscriptions_charge_intents" FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "standard_invoice_settlements" table
ALTER TABLE "standard_invoice_settlements" ADD CONSTRAINT "standard_invoice_settlements_billing_invoice_lines_billing_invo" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD CONSTRAINT "standard_invoice_settlements_charges_standard_invoice_settlment" FOREIGN KEY ("charge_id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
