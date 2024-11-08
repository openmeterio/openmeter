-- reverse: drop "billing_invoice_manual_usage_based_line_configs" table
CREATE TABLE "billing_invoice_manual_usage_based_line_configs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "price_type" character varying NOT NULL,
  "feature_key" character varying NOT NULL,
  "price" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "billinginvoicemanualusagebasedlineconfig_id" ON "billing_invoice_manual_usage_based_line_configs" ("id");
CREATE INDEX "billinginvoicemanualusagebasedlineconfig_namespace" ON "billing_invoice_manual_usage_based_line_configs" ("namespace");
-- reverse: drop "billing_invoice_manual_line_configs" table
CREATE TABLE "billing_invoice_manual_line_configs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "unit_price" numeric NOT NULL,
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "billinginvoicemanuallineconfig_id" ON "billing_invoice_manual_line_configs" ("id");
CREATE INDEX "billinginvoicemanuallineconfig_namespace" ON "billing_invoice_manual_line_configs" ("namespace");
-- reverse: rename a column from "manual_usage_based_line_config_id" to "usage_based_line_config_id"
ALTER TABLE "billing_invoice_lines" RENAME COLUMN "usage_based_line_config_id" TO "manual_usage_based_line_config_id";
-- reverse: rename a column from "manual_line_config_id" to "fee_line_config_id"
ALTER TABLE "billing_invoice_lines" RENAME COLUMN "fee_line_config_id" TO "manual_line_config_id";
-- reverse: rename a column from "tax_overrides" to "tax_config"
ALTER TABLE "billing_invoice_lines" RENAME COLUMN "tax_config" TO "tax_overrides";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_billing_invoice_usage_based_line_configs_", DROP CONSTRAINT "billing_invoice_lines_billing_invoice_flat_fee_line_configs_fla", ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_manual_usage_based_line_c" FOREIGN KEY ("manual_usage_based_line_config_id") REFERENCES "billing_invoice_manual_usage_based_line_configs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_manual_line_configs_manua" FOREIGN KEY ("manual_line_config_id") REFERENCES "billing_invoice_manual_line_configs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- reverse: create index "billinginvoiceusagebasedlineconfig_namespace" to table: "billing_invoice_usage_based_line_configs"
DROP INDEX "billinginvoiceusagebasedlineconfig_namespace";
-- reverse: create index "billinginvoiceusagebasedlineconfig_id" to table: "billing_invoice_usage_based_line_configs"
DROP INDEX "billinginvoiceusagebasedlineconfig_id";
-- reverse: create "billing_invoice_usage_based_line_configs" table
DROP TABLE "billing_invoice_usage_based_line_configs";
-- reverse: create index "billinginvoiceflatfeelineconfig_namespace" to table: "billing_invoice_flat_fee_line_configs"
DROP INDEX "billinginvoiceflatfeelineconfig_namespace";
-- reverse: create index "billinginvoiceflatfeelineconfig_id" to table: "billing_invoice_flat_fee_line_configs"
DROP INDEX "billinginvoiceflatfeelineconfig_id";
-- reverse: create "billing_invoice_flat_fee_line_configs" table
DROP TABLE "billing_invoice_flat_fee_line_configs";
