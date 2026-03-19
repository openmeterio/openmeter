-- modify "charge_flat_fee_payments" table
-- atlas:nolint MF103
ALTER TABLE "charge_flat_fee_payments" ADD COLUMN "invoice_id" character(26) NOT NULL;
-- modify "charge_usage_based_run_payments" table
-- atlas:nolint MF103
ALTER TABLE "charge_usage_based_run_payments" ADD COLUMN "invoice_id" character(26) NOT NULL;
-- create "charge_credit_purchase_invoiced_payments" table
CREATE TABLE "charge_credit_purchase_invoiced_payments" (
  "id" character(26) NOT NULL,
  "invoice_id" character(26) NOT NULL,
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
  CONSTRAINT "charge_credit_purchase_invoiced_payments_billing_invoice_lines_" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "charge_credit_purchase_invoiced_payments_charge_credit_purchase" FOREIGN KEY ("charge_id") REFERENCES "charge_credit_purchases" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "charge_credit_purchase_invoiced_payments_charge_id_key" to table: "charge_credit_purchase_invoiced_payments"
CREATE UNIQUE INDEX "charge_credit_purchase_invoiced_payments_charge_id_key" ON "charge_credit_purchase_invoiced_payments" ("charge_id");
-- create index "charge_credit_purchase_invoiced_payments_line_id_key" to table: "charge_credit_purchase_invoiced_payments"
CREATE UNIQUE INDEX "charge_credit_purchase_invoiced_payments_line_id_key" ON "charge_credit_purchase_invoiced_payments" ("line_id");
-- create index "chargecreditpurchaseinvoicedpayment_annotations" to table: "charge_credit_purchase_invoiced_payments"
CREATE INDEX "chargecreditpurchaseinvoicedpayment_annotations" ON "charge_credit_purchase_invoiced_payments" USING gin ("annotations");
-- create index "chargecreditpurchaseinvoicedpayment_id" to table: "charge_credit_purchase_invoiced_payments"
CREATE UNIQUE INDEX "chargecreditpurchaseinvoicedpayment_id" ON "charge_credit_purchase_invoiced_payments" ("id");
-- create index "chargecreditpurchaseinvoicedpayment_namespace" to table: "charge_credit_purchase_invoiced_payments"
CREATE INDEX "chargecreditpurchaseinvoicedpayment_namespace" ON "charge_credit_purchase_invoiced_payments" ("namespace");
-- create index "chargecreditpurchaseinvoicedpayment_namespace_charge_id" to table: "charge_credit_purchase_invoiced_payments"
CREATE UNIQUE INDEX "chargecreditpurchaseinvoicedpayment_namespace_charge_id" ON "charge_credit_purchase_invoiced_payments" ("namespace", "charge_id");
