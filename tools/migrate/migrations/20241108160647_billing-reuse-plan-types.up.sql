-- create "billing_invoice_flat_fee_line_configs" table
CREATE TABLE "billing_invoice_flat_fee_line_configs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "amount" numeric NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "billinginvoiceflatfeelineconfig_id" to table: "billing_invoice_flat_fee_line_configs"
CREATE UNIQUE INDEX "billinginvoiceflatfeelineconfig_id" ON "billing_invoice_flat_fee_line_configs" ("id");
-- create index "billinginvoiceflatfeelineconfig_namespace" to table: "billing_invoice_flat_fee_line_configs"
CREATE INDEX "billinginvoiceflatfeelineconfig_namespace" ON "billing_invoice_flat_fee_line_configs" ("namespace");
-- create "billing_invoice_usage_based_line_configs" table
CREATE TABLE "billing_invoice_usage_based_line_configs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "price_type" character varying NOT NULL,
  "feature_key" character varying NOT NULL,
  "price" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
-- rename a column from "tax_overrides" to "tax_config"
-- atlas:nolint BC102
ALTER TABLE "billing_invoice_lines" RENAME COLUMN "tax_overrides" TO "tax_config";
-- rename a column from "manual_line_config_id" to "fee_line_config_id"
-- atlas:nolint BC102
ALTER TABLE "billing_invoice_lines" RENAME COLUMN "manual_line_config_id" TO "fee_line_config_id";
-- rename a column from "manual_usage_based_line_config_id" to "usage_based_line_config_id"
-- atlas:nolint BC102
ALTER TABLE "billing_invoice_lines" RENAME COLUMN "manual_usage_based_line_config_id" TO "usage_based_line_config_id";
-- create index "billinginvoiceusagebasedlineconfig_id" to table: "billing_invoice_usage_based_line_configs"
CREATE UNIQUE INDEX "billinginvoiceusagebasedlineconfig_id" ON "billing_invoice_usage_based_line_configs" ("id");
-- create index "billinginvoiceusagebasedlineconfig_namespace" to table: "billing_invoice_usage_based_line_configs"
CREATE INDEX "billinginvoiceusagebasedlineconfig_namespace" ON "billing_invoice_usage_based_line_configs" ("namespace");
-- modify "billing_invoice_lines" table
-- atlas:nolint CD101
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_billing_invoice_manual_line_configs_manua", DROP CONSTRAINT "billing_invoice_lines_billing_invoice_manual_usage_based_line_c", ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_flat_fee_line_configs_fla" FOREIGN KEY ("fee_line_config_id") REFERENCES "billing_invoice_flat_fee_line_configs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_usage_based_line_configs_" FOREIGN KEY ("usage_based_line_config_id") REFERENCES "billing_invoice_usage_based_line_configs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- drop "billing_invoice_manual_line_configs" table
-- atlas:nolint DS102
DROP TABLE "billing_invoice_manual_line_configs";
-- drop "billing_invoice_manual_usage_based_line_configs" table
-- atlas:nolint DS102
DROP TABLE "billing_invoice_manual_usage_based_line_configs";
