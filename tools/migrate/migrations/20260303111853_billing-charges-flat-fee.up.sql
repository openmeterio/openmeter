-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP COLUMN "billing_invoice_line_standard_invoice_settlments";
-- create "charge_credit_realizations" table
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
-- create index "chargecreditrealization_annotations" to table: "charge_credit_realizations"
CREATE INDEX "chargecreditrealization_annotations" ON "charge_credit_realizations" USING gin ("annotations");
-- create index "chargecreditrealization_id" to table: "charge_credit_realizations"
CREATE UNIQUE INDEX "chargecreditrealization_id" ON "charge_credit_realizations" ("id");
-- create index "chargecreditrealization_namespace" to table: "charge_credit_realizations"
CREATE INDEX "chargecreditrealization_namespace" ON "charge_credit_realizations" ("namespace");
-- create "charge_standard_invoice_accrued_usages" table
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
-- create index "charge_standard_invoice_accrued_usages_charge_id_key" to table: "charge_standard_invoice_accrued_usages"
CREATE UNIQUE INDEX "charge_standard_invoice_accrued_usages_charge_id_key" ON "charge_standard_invoice_accrued_usages" ("charge_id");
-- create index "chargestandardinvoiceaccruedusage_annotations" to table: "charge_standard_invoice_accrued_usages"
CREATE INDEX "chargestandardinvoiceaccruedusage_annotations" ON "charge_standard_invoice_accrued_usages" USING gin ("annotations");
-- create index "chargestandardinvoiceaccruedusage_id" to table: "charge_standard_invoice_accrued_usages"
CREATE UNIQUE INDEX "chargestandardinvoiceaccruedusage_id" ON "charge_standard_invoice_accrued_usages" ("id");
-- create index "chargestandardinvoiceaccruedusage_namespace" to table: "charge_standard_invoice_accrued_usages"
CREATE INDEX "chargestandardinvoiceaccruedusage_namespace" ON "charge_standard_invoice_accrued_usages" ("namespace");
-- create index "chargestandardinvoiceaccruedusage_namespace_charge_id_line_id" to table: "charge_standard_invoice_accrued_usages"
CREATE UNIQUE INDEX "chargestandardinvoiceaccruedusage_namespace_charge_id_line_id" ON "charge_standard_invoice_accrued_usages" ("namespace", "charge_id", "line_id") WHERE ((line_id IS NOT NULL) AND (deleted_at IS NULL));
-- create "charge_standard_invoice_payment_settlements" table
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
  "charge_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "charge_standard_invoice_payment_settlements_billing_invoice_lin" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "charge_standard_invoice_payment_settlements_charge_flat_fees_ch" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "charge_standard_invoice_payment_settlements_charge_id_key" to table: "charge_standard_invoice_payment_settlements"
CREATE UNIQUE INDEX "charge_standard_invoice_payment_settlements_charge_id_key" ON "charge_standard_invoice_payment_settlements" ("charge_id");
-- create index "charge_standard_invoice_payment_settlements_line_id_key" to table: "charge_standard_invoice_payment_settlements"
CREATE UNIQUE INDEX "charge_standard_invoice_payment_settlements_line_id_key" ON "charge_standard_invoice_payment_settlements" ("line_id");
-- create index "chargestandardinvoicepaymentsettlement_annotations" to table: "charge_standard_invoice_payment_settlements"
CREATE INDEX "chargestandardinvoicepaymentsettlement_annotations" ON "charge_standard_invoice_payment_settlements" USING gin ("annotations");
-- create index "chargestandardinvoicepaymentsettlement_id" to table: "charge_standard_invoice_payment_settlements"
CREATE UNIQUE INDEX "chargestandardinvoicepaymentsettlement_id" ON "charge_standard_invoice_payment_settlements" ("id");
-- create index "chargestandardinvoicepaymentsettlement_namespace" to table: "charge_standard_invoice_payment_settlements"
CREATE INDEX "chargestandardinvoicepaymentsettlement_namespace" ON "charge_standard_invoice_payment_settlements" ("namespace");
-- create index "chargestandardinvoicepaymentsettlement_namespace_charge_id_line" to table: "charge_standard_invoice_payment_settlements"
CREATE UNIQUE INDEX "chargestandardinvoicepaymentsettlement_namespace_charge_id_line" ON "charge_standard_invoice_payment_settlements" ("namespace", "charge_id", "line_id") WHERE ((line_id IS NOT NULL) AND (deleted_at IS NULL));
-- drop "standard_invoice_settlements" table
DROP TABLE "standard_invoice_settlements";
