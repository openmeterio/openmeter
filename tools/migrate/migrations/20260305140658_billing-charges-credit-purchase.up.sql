-- create "charge_external_payment_settlements" table
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
-- create index "chargeexternalpaymentsettlement_annotations" to table: "charge_external_payment_settlements"
CREATE INDEX "chargeexternalpaymentsettlement_annotations" ON "charge_external_payment_settlements" USING gin ("annotations");
-- create index "chargeexternalpaymentsettlement_id" to table: "charge_external_payment_settlements"
CREATE UNIQUE INDEX "chargeexternalpaymentsettlement_id" ON "charge_external_payment_settlements" ("id");
-- create index "chargeexternalpaymentsettlement_namespace" to table: "charge_external_payment_settlements"
CREATE INDEX "chargeexternalpaymentsettlement_namespace" ON "charge_external_payment_settlements" ("namespace");
-- create index "chargeexternalpaymentsettlement_namespace_id" to table: "charge_external_payment_settlements"
CREATE UNIQUE INDEX "chargeexternalpaymentsettlement_namespace_id" ON "charge_external_payment_settlements" ("namespace", "id");
-- modify "charge_credit_purchases" table
ALTER TABLE "charge_credit_purchases" DROP COLUMN "status", ADD COLUMN "credit_grant_transaction_group_id" character(26) NULL, ADD COLUMN "credit_granted_at" timestamptz NULL, ADD COLUMN "external_payment_settlement_id" character(26) NULL, ADD CONSTRAINT "charge_credit_purchases_charge_external_payment_settlements_cha" FOREIGN KEY ("external_payment_settlement_id") REFERENCES "charge_external_payment_settlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "charge_credit_purchases_external_payment_settlement_id_key" to table: "charge_credit_purchases"
CREATE UNIQUE INDEX "charge_credit_purchases_external_payment_settlement_id_key" ON "charge_credit_purchases" ("external_payment_settlement_id");
-- modify "charge_standard_invoice_payment_settlements" table
ALTER TABLE "charge_standard_invoice_payment_settlements" DROP COLUMN "charge_id";
-- create index "chargestandardinvoicepaymentsettlement_namespace_line_id" to table: "charge_standard_invoice_payment_settlements"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "chargestandardinvoicepaymentsettlement_namespace_line_id" ON "charge_standard_invoice_payment_settlements" ("namespace", "line_id") WHERE ((line_id IS NOT NULL) AND (deleted_at IS NULL));
-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "std_invoice_payment_settlement_id" character(26) NULL, ADD CONSTRAINT "charge_flat_fees_charge_standard_invoice_payment_settlements_ch" FOREIGN KEY ("std_invoice_payment_settlement_id") REFERENCES "charge_standard_invoice_payment_settlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "charge_flat_fees_std_invoice_payment_settlement_id_key" to table: "charge_flat_fees"
CREATE UNIQUE INDEX "charge_flat_fees_std_invoice_payment_settlement_id_key" ON "charge_flat_fees" ("std_invoice_payment_settlement_id");
