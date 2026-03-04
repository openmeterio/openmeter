-- reverse: drop "standard_invoice_settlements" table
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
  "credits_total" numeric NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "standard_invoice_settlements_billing_invoice_lines_billing_invo" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "standard_invoice_settlements_charges_standard_invoice_settlment" FOREIGN KEY ("charge_id") REFERENCES "charges" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
CREATE INDEX "standardinvoicesettlement_annotations" ON "standard_invoice_settlements" USING gin ("annotations");
CREATE UNIQUE INDEX "standardinvoicesettlement_id" ON "standard_invoice_settlements" ("id");
CREATE INDEX "standardinvoicesettlement_namespace" ON "standard_invoice_settlements" ("namespace");
CREATE UNIQUE INDEX "standardinvoicesettlement_namespace_charge_id_line_id" ON "standard_invoice_settlements" ("namespace", "charge_id", "line_id");
-- reverse: create index "chargestandardinvoicepaymentsettlement_namespace_charge_id_line" to table: "charge_standard_invoice_payment_settlements"
DROP INDEX "chargestandardinvoicepaymentsettlement_namespace_charge_id_line";
-- reverse: create index "chargestandardinvoicepaymentsettlement_namespace" to table: "charge_standard_invoice_payment_settlements"
DROP INDEX "chargestandardinvoicepaymentsettlement_namespace";
-- reverse: create index "chargestandardinvoicepaymentsettlement_id" to table: "charge_standard_invoice_payment_settlements"
DROP INDEX "chargestandardinvoicepaymentsettlement_id";
-- reverse: create index "chargestandardinvoicepaymentsettlement_annotations" to table: "charge_standard_invoice_payment_settlements"
DROP INDEX "chargestandardinvoicepaymentsettlement_annotations";
-- reverse: create index "charge_standard_invoice_payment_settlements_line_id_key" to table: "charge_standard_invoice_payment_settlements"
DROP INDEX "charge_standard_invoice_payment_settlements_line_id_key";
-- reverse: create index "charge_standard_invoice_payment_settlements_charge_id_key" to table: "charge_standard_invoice_payment_settlements"
DROP INDEX "charge_standard_invoice_payment_settlements_charge_id_key";
-- reverse: create "charge_standard_invoice_payment_settlements" table
DROP TABLE "charge_standard_invoice_payment_settlements";
-- reverse: create index "chargestandardinvoiceaccruedusage_namespace_charge_id_line_id" to table: "charge_standard_invoice_accrued_usages"
DROP INDEX "chargestandardinvoiceaccruedusage_namespace_charge_id_line_id";
-- reverse: create index "chargestandardinvoiceaccruedusage_namespace" to table: "charge_standard_invoice_accrued_usages"
DROP INDEX "chargestandardinvoiceaccruedusage_namespace";
-- reverse: create index "chargestandardinvoiceaccruedusage_id" to table: "charge_standard_invoice_accrued_usages"
DROP INDEX "chargestandardinvoiceaccruedusage_id";
-- reverse: create index "chargestandardinvoiceaccruedusage_annotations" to table: "charge_standard_invoice_accrued_usages"
DROP INDEX "chargestandardinvoiceaccruedusage_annotations";
-- reverse: create index "charge_standard_invoice_accrued_usages_charge_id_key" to table: "charge_standard_invoice_accrued_usages"
DROP INDEX "charge_standard_invoice_accrued_usages_charge_id_key";
-- reverse: create "charge_standard_invoice_accrued_usages" table
DROP TABLE "charge_standard_invoice_accrued_usages";
-- reverse: create index "chargecreditrealization_namespace" to table: "charge_credit_realizations"
DROP INDEX "chargecreditrealization_namespace";
-- reverse: create index "chargecreditrealization_id" to table: "charge_credit_realizations"
DROP INDEX "chargecreditrealization_id";
-- reverse: create index "chargecreditrealization_annotations" to table: "charge_credit_realizations"
DROP INDEX "chargecreditrealization_annotations";
-- reverse: create "charge_credit_realizations" table
DROP TABLE "charge_credit_realizations";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "billing_invoice_line_standard_invoice_settlments" character(26) NULL;
ALTER TABLE "billing_invoice_lines" ADD CONSTRAINT "billing_invoice_lines_standard_invoice_settlements_standard_inv" FOREIGN KEY ("billing_invoice_line_standard_invoice_settlments") REFERENCES "standard_invoice_settlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
