-- modify "charge_credit_purchases" table
-- atlas:nolint DS103
ALTER TABLE "charge_credit_purchases" DROP COLUMN "external_payment_settlement_id";
-- create "charge_credit_purchase_external_payments" table
CREATE TABLE "charge_credit_purchase_external_payments" (
  "id" character(26) NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "amount" numeric NOT NULL,
  "authorized_transaction_group_id" character(26) NULL,
  "authorized_at" timestamptz NULL,
  "settled_transaction_group_id" character(26) NULL,
  "settled_at" timestamptz NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_credit_purchase_external_payments_charge_credit_purchase" FOREIGN KEY ("charge_id") REFERENCES "charge_credit_purchases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "charge_credit_purchase_external_payments_charge_id_key" to table: "charge_credit_purchase_external_payments"
CREATE UNIQUE INDEX "charge_credit_purchase_external_payments_charge_id_key" ON "charge_credit_purchase_external_payments" ("charge_id");
-- create index "chargecreditpurchaseexternalpayment_annotations" to table: "charge_credit_purchase_external_payments"
CREATE INDEX "chargecreditpurchaseexternalpayment_annotations" ON "charge_credit_purchase_external_payments" USING gin ("annotations");
-- create index "chargecreditpurchaseexternalpayment_id" to table: "charge_credit_purchase_external_payments"
CREATE UNIQUE INDEX "chargecreditpurchaseexternalpayment_id" ON "charge_credit_purchase_external_payments" ("id");
-- create index "chargecreditpurchaseexternalpayment_namespace" to table: "charge_credit_purchase_external_payments"
CREATE INDEX "chargecreditpurchaseexternalpayment_namespace" ON "charge_credit_purchase_external_payments" ("namespace");
-- create index "chargecreditpurchaseexternalpayment_namespace_charge_id" to table: "charge_credit_purchase_external_payments"
CREATE UNIQUE INDEX "chargecreditpurchaseexternalpayment_namespace_charge_id" ON "charge_credit_purchase_external_payments" ("namespace", "charge_id");
-- modify "charge_flat_fees" table
-- atlas:nolint DS103
ALTER TABLE "charge_flat_fees" DROP COLUMN "std_invoice_payment_settlement_id";
-- create "charge_flat_fee_credit_allocations" table
CREATE TABLE "charge_flat_fee_credit_allocations" (
  "id" character(26) NOT NULL,
  "amount" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "ledger_transaction_group_id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "line_id" character(26) NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_flat_fee_credit_allocations_billing_invoice_lines_charge" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "charge_flat_fee_credit_allocations_charge_flat_fees_credit_allo" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "chargeflatfeecreditallocations_annotations" to table: "charge_flat_fee_credit_allocations"
CREATE INDEX "chargeflatfeecreditallocations_annotations" ON "charge_flat_fee_credit_allocations" USING gin ("annotations");
-- create index "chargeflatfeecreditallocations_id" to table: "charge_flat_fee_credit_allocations"
CREATE UNIQUE INDEX "chargeflatfeecreditallocations_id" ON "charge_flat_fee_credit_allocations" ("id");
-- create index "chargeflatfeecreditallocations_namespace" to table: "charge_flat_fee_credit_allocations"
CREATE INDEX "chargeflatfeecreditallocations_namespace" ON "charge_flat_fee_credit_allocations" ("namespace");
-- create index "chargeflatfeecreditallocations_namespace_charge_id_line_id_dele" to table: "charge_flat_fee_credit_allocations"
CREATE UNIQUE INDEX "chargeflatfeecreditallocations_namespace_charge_id_line_id_dele" ON "charge_flat_fee_credit_allocations" ("namespace", "charge_id", "line_id", "deleted_at") WHERE ((line_id IS NOT NULL) AND (deleted_at IS NULL));
-- create "charge_flat_fee_invoiced_usages" table
CREATE TABLE "charge_flat_fee_invoiced_usages" (
  "id" character(26) NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "mutable" boolean NOT NULL,
  "ledger_transaction_group_id" character(26) NULL,
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
  "line_id" character(26) NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_flat_fee_invoiced_usages_billing_invoice_lines_charge_fl" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "charge_flat_fee_invoiced_usages_charge_flat_fees_invoiced_usage" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "charge_flat_fee_invoiced_usages_charge_id_key" to table: "charge_flat_fee_invoiced_usages"
CREATE UNIQUE INDEX "charge_flat_fee_invoiced_usages_charge_id_key" ON "charge_flat_fee_invoiced_usages" ("charge_id");
-- create index "chargeflatfeeinvoicedusage_annotations" to table: "charge_flat_fee_invoiced_usages"
CREATE INDEX "chargeflatfeeinvoicedusage_annotations" ON "charge_flat_fee_invoiced_usages" USING gin ("annotations");
-- create index "chargeflatfeeinvoicedusage_id" to table: "charge_flat_fee_invoiced_usages"
CREATE UNIQUE INDEX "chargeflatfeeinvoicedusage_id" ON "charge_flat_fee_invoiced_usages" ("id");
-- create index "chargeflatfeeinvoicedusage_namespace" to table: "charge_flat_fee_invoiced_usages"
CREATE INDEX "chargeflatfeeinvoicedusage_namespace" ON "charge_flat_fee_invoiced_usages" ("namespace");
-- create index "chargeflatfeeinvoicedusage_namespace_charge_id" to table: "charge_flat_fee_invoiced_usages"
CREATE UNIQUE INDEX "chargeflatfeeinvoicedusage_namespace_charge_id" ON "charge_flat_fee_invoiced_usages" ("namespace", "charge_id");
-- create "charge_flat_fee_payments" table
CREATE TABLE "charge_flat_fee_payments" (
  "id" character(26) NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "amount" numeric NOT NULL,
  "authorized_transaction_group_id" character(26) NULL,
  "authorized_at" timestamptz NULL,
  "settled_transaction_group_id" character(26) NULL,
  "settled_at" timestamptz NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "annotations" jsonb NULL,
  "line_id" character(26) NOT NULL,
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_flat_fee_payments_billing_invoice_lines_charge_flat_fee_" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "charge_flat_fee_payments_charge_flat_fees_payment" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "charge_flat_fee_payments_charge_id_key" to table: "charge_flat_fee_payments"
CREATE UNIQUE INDEX "charge_flat_fee_payments_charge_id_key" ON "charge_flat_fee_payments" ("charge_id");
-- create index "charge_flat_fee_payments_line_id_key" to table: "charge_flat_fee_payments"
CREATE UNIQUE INDEX "charge_flat_fee_payments_line_id_key" ON "charge_flat_fee_payments" ("line_id");
-- create index "chargeflatfeepayment_annotations" to table: "charge_flat_fee_payments"
CREATE INDEX "chargeflatfeepayment_annotations" ON "charge_flat_fee_payments" USING gin ("annotations");
-- create index "chargeflatfeepayment_id" to table: "charge_flat_fee_payments"
CREATE UNIQUE INDEX "chargeflatfeepayment_id" ON "charge_flat_fee_payments" ("id");
-- create index "chargeflatfeepayment_namespace" to table: "charge_flat_fee_payments"
CREATE INDEX "chargeflatfeepayment_namespace" ON "charge_flat_fee_payments" ("namespace");
-- create index "chargeflatfeepayment_namespace_charge_id" to table: "charge_flat_fee_payments"
CREATE UNIQUE INDEX "chargeflatfeepayment_namespace_charge_id" ON "charge_flat_fee_payments" ("namespace", "charge_id");
-- drop "charge_credit_realizations" table
-- atlas:nolint DS102
DROP TABLE "charge_credit_realizations";
-- drop "charge_standard_invoice_accrued_usages" table
-- atlas:nolint DS102
DROP TABLE "charge_standard_invoice_accrued_usages";
-- drop "charge_usage_baseds" table
-- atlas:nolint DS102
DROP TABLE "charge_usage_baseds";
-- drop "charge_external_payment_settlements" table
-- atlas:nolint DS102
DROP TABLE "charge_external_payment_settlements";
-- drop "charge_standard_invoice_payment_settlements" table
-- atlas:nolint DS102
DROP TABLE "charge_standard_invoice_payment_settlements";
