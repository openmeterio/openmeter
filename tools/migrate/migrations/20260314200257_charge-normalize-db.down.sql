-- reverse: drop "charge_standard_invoice_payment_settlements" table
CREATE TABLE "charge_standard_invoice_payment_settlements" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "amount" numeric NOT NULL,
  "authorized_transaction_group_id" character(26) NULL,
  "authorized_at" timestamptz NULL,
  "settled_transaction_group_id" character(26) NULL,
  "settled_at" timestamptz NULL,
  "line_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_standard_invoice_payment_settlements_billing_invoice_lin" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
CREATE UNIQUE INDEX "charge_standard_invoice_payment_settlements_line_id_key" ON "charge_standard_invoice_payment_settlements" ("line_id");
CREATE INDEX "chargestandardinvoicepaymentsettlement_annotations" ON "charge_standard_invoice_payment_settlements" USING gin ("annotations");
CREATE UNIQUE INDEX "chargestandardinvoicepaymentsettlement_id" ON "charge_standard_invoice_payment_settlements" ("id");
CREATE INDEX "chargestandardinvoicepaymentsettlement_namespace" ON "charge_standard_invoice_payment_settlements" ("namespace");
CREATE UNIQUE INDEX "chargestandardinvoicepaymentsettlement_namespace_line_id" ON "charge_standard_invoice_payment_settlements" ("namespace", "line_id") WHERE ((line_id IS NOT NULL) AND (deleted_at IS NULL));
-- reverse: drop "charge_external_payment_settlements" table
CREATE TABLE "charge_external_payment_settlements" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "amount" numeric NOT NULL,
  "authorized_transaction_group_id" character(26) NULL,
  "authorized_at" timestamptz NULL,
  "settled_transaction_group_id" character(26) NULL,
  "settled_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
CREATE INDEX "chargeexternalpaymentsettlement_annotations" ON "charge_external_payment_settlements" USING gin ("annotations");
CREATE UNIQUE INDEX "chargeexternalpaymentsettlement_id" ON "charge_external_payment_settlements" ("id");
CREATE INDEX "chargeexternalpaymentsettlement_namespace" ON "charge_external_payment_settlements" ("namespace");
CREATE UNIQUE INDEX "chargeexternalpaymentsettlement_namespace_id" ON "charge_external_payment_settlements" ("namespace", "id");
-- reverse: drop "charge_usage_baseds" table
CREATE TABLE "charge_usage_baseds" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "price" jsonb NOT NULL,
  "feature_key" character varying NOT NULL,
  "invoice_at" timestamptz NOT NULL,
  "settlement_mode" character varying NOT NULL,
  "discounts" jsonb NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_usage_baseds_charges_usage_based" FOREIGN KEY ("id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
CREATE UNIQUE INDEX "chargeusagebased_id" ON "charge_usage_baseds" ("id");
CREATE INDEX "chargeusagebased_namespace" ON "charge_usage_baseds" ("namespace");
CREATE UNIQUE INDEX "chargeusagebased_namespace_id" ON "charge_usage_baseds" ("namespace", "id");
-- reverse: drop "charge_standard_invoice_accrued_usages" table
CREATE TABLE "charge_standard_invoice_accrued_usages" (
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
  "credits_total" numeric NOT NULL,
  "total" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "mutable" boolean NOT NULL,
  "ledger_transaction_group_id" character(26) NULL,
  "line_id" character(26) NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_standard_invoice_accrued_usages_billing_invoice_lines_ch" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "charge_standard_invoice_accrued_usages_charge_flat_fees_charge_" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
CREATE UNIQUE INDEX "charge_standard_invoice_accrued_usages_charge_id_key" ON "charge_standard_invoice_accrued_usages" ("charge_id");
CREATE INDEX "chargestandardinvoiceaccruedusage_annotations" ON "charge_standard_invoice_accrued_usages" USING gin ("annotations");
CREATE UNIQUE INDEX "chargestandardinvoiceaccruedusage_id" ON "charge_standard_invoice_accrued_usages" ("id");
CREATE INDEX "chargestandardinvoiceaccruedusage_namespace" ON "charge_standard_invoice_accrued_usages" ("namespace");
CREATE UNIQUE INDEX "chargestandardinvoiceaccruedusage_namespace_charge_id_line_id" ON "charge_standard_invoice_accrued_usages" ("namespace", "charge_id", "line_id") WHERE ((line_id IS NOT NULL) AND (deleted_at IS NULL));
-- reverse: drop "charge_credit_realizations" table
CREATE TABLE "charge_credit_realizations" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "amount" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "ledger_transaction_group_id" character(26) NOT NULL,
  "line_id" character(26) NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_credit_realizations_billing_invoice_lines_charge_credit_" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "charge_credit_realizations_charge_flat_fees_charge_credit_reali" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
CREATE INDEX "chargecreditrealization_annotations" ON "charge_credit_realizations" USING gin ("annotations");
CREATE UNIQUE INDEX "chargecreditrealization_id" ON "charge_credit_realizations" ("id");
CREATE INDEX "chargecreditrealization_namespace" ON "charge_credit_realizations" ("namespace");
-- reverse: create index "chargeflatfeepayment_namespace_charge_id" to table: "charge_flat_fee_payments"
DROP INDEX "chargeflatfeepayment_namespace_charge_id";
-- reverse: create index "chargeflatfeepayment_namespace" to table: "charge_flat_fee_payments"
DROP INDEX "chargeflatfeepayment_namespace";
-- reverse: create index "chargeflatfeepayment_id" to table: "charge_flat_fee_payments"
DROP INDEX "chargeflatfeepayment_id";
-- reverse: create index "chargeflatfeepayment_annotations" to table: "charge_flat_fee_payments"
DROP INDEX "chargeflatfeepayment_annotations";
-- reverse: create index "charge_flat_fee_payments_line_id_key" to table: "charge_flat_fee_payments"
DROP INDEX "charge_flat_fee_payments_line_id_key";
-- reverse: create index "charge_flat_fee_payments_charge_id_key" to table: "charge_flat_fee_payments"
DROP INDEX "charge_flat_fee_payments_charge_id_key";
-- reverse: create "charge_flat_fee_payments" table
DROP TABLE "charge_flat_fee_payments";
-- reverse: create index "chargeflatfeeinvoicedusage_namespace_charge_id" to table: "charge_flat_fee_invoiced_usages"
DROP INDEX "chargeflatfeeinvoicedusage_namespace_charge_id";
-- reverse: create index "chargeflatfeeinvoicedusage_namespace" to table: "charge_flat_fee_invoiced_usages"
DROP INDEX "chargeflatfeeinvoicedusage_namespace";
-- reverse: create index "chargeflatfeeinvoicedusage_id" to table: "charge_flat_fee_invoiced_usages"
DROP INDEX "chargeflatfeeinvoicedusage_id";
-- reverse: create index "chargeflatfeeinvoicedusage_annotations" to table: "charge_flat_fee_invoiced_usages"
DROP INDEX "chargeflatfeeinvoicedusage_annotations";
-- reverse: create index "charge_flat_fee_invoiced_usages_charge_id_key" to table: "charge_flat_fee_invoiced_usages"
DROP INDEX "charge_flat_fee_invoiced_usages_charge_id_key";
-- reverse: create "charge_flat_fee_invoiced_usages" table
DROP TABLE "charge_flat_fee_invoiced_usages";
-- reverse: create index "chargeflatfeecreditallocations_namespace_charge_id_line_id_dele" to table: "charge_flat_fee_credit_allocations"
DROP INDEX "chargeflatfeecreditallocations_namespace_charge_id_line_id_dele";
-- reverse: create index "chargeflatfeecreditallocations_namespace" to table: "charge_flat_fee_credit_allocations"
DROP INDEX "chargeflatfeecreditallocations_namespace";
-- reverse: create index "chargeflatfeecreditallocations_id" to table: "charge_flat_fee_credit_allocations"
DROP INDEX "chargeflatfeecreditallocations_id";
-- reverse: create index "chargeflatfeecreditallocations_annotations" to table: "charge_flat_fee_credit_allocations"
DROP INDEX "chargeflatfeecreditallocations_annotations";
-- reverse: create "charge_flat_fee_credit_allocations" table
DROP TABLE "charge_flat_fee_credit_allocations";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "std_invoice_payment_settlement_id" character(26) NULL;
-- reverse: create index "chargecreditpurchaseexternalpayment_namespace_charge_id" to table: "charge_credit_purchase_external_payments"
DROP INDEX "chargecreditpurchaseexternalpayment_namespace_charge_id";
-- reverse: create index "chargecreditpurchaseexternalpayment_namespace" to table: "charge_credit_purchase_external_payments"
DROP INDEX "chargecreditpurchaseexternalpayment_namespace";
-- reverse: create index "chargecreditpurchaseexternalpayment_id" to table: "charge_credit_purchase_external_payments"
DROP INDEX "chargecreditpurchaseexternalpayment_id";
-- reverse: create index "chargecreditpurchaseexternalpayment_annotations" to table: "charge_credit_purchase_external_payments"
DROP INDEX "chargecreditpurchaseexternalpayment_annotations";
-- reverse: create index "charge_credit_purchase_external_payments_charge_id_key" to table: "charge_credit_purchase_external_payments"
DROP INDEX "charge_credit_purchase_external_payments_charge_id_key";
-- reverse: create "charge_credit_purchase_external_payments" table
DROP TABLE "charge_credit_purchase_external_payments";
-- reverse: modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" ADD COLUMN "external_payment_settlement_id" character(26) NULL;

-- reverse fixes
-- create index "charge_flat_fees_std_invoice_payment_settlement_id_key" to table: "charge_flat_fees"
CREATE UNIQUE INDEX "charge_flat_fees_std_invoice_payment_settlement_id_key" ON "charge_flat_fees" ("std_invoice_payment_settlement_id");
ALTER TABLE "charge_flat_fees"  ADD CONSTRAINT "charge_flat_fees_charge_standard_invoice_payment_settlements_ch" FOREIGN KEY ("std_invoice_payment_settlement_id") REFERENCES "charge_standard_invoice_payment_settlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;

CREATE UNIQUE INDEX "charge_credit_purchases_external_payment_settlement_id_key" ON "charge_credit_purchases" ("external_payment_settlement_id");
ALTER TABLE "charge_credit_purchases" ADD CONSTRAINT "charge_credit_purchases_charge_external_payment_settlements_cha" FOREIGN KEY ("external_payment_settlement_id") REFERENCES "charge_external_payment_settlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
