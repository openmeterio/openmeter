-- modify "billing_invoices" table
-- atlas:nolint MF103
ALTER TABLE "billing_invoices" ADD COLUMN "customer_usage_attribution" jsonb NOT NULL;
-- create "billing_invoice_manual_usage_based_line_configs" table
CREATE TABLE "billing_invoice_manual_usage_based_line_configs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "price_type" character varying NOT NULL,
  "feature_key" character varying NOT NULL,
  "price" jsonb NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "billinginvoicemanualusagebasedlineconfig_id" to table: "billing_invoice_manual_usage_based_line_configs"
CREATE UNIQUE INDEX "billinginvoicemanualusagebasedlineconfig_id" ON "billing_invoice_manual_usage_based_line_configs" ("id");
-- create index "billinginvoicemanualusagebasedlineconfig_namespace" to table: "billing_invoice_manual_usage_based_line_configs"
CREATE INDEX "billinginvoicemanualusagebasedlineconfig_namespace" ON "billing_invoice_manual_usage_based_line_configs" ("namespace");
-- modify "billing_invoice_lines" table
-- atlas:nolint CD101
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_billing_invoice_manual_line_configs_billi", DROP CONSTRAINT "billing_invoice_lines_billing_invoices_billing_invoice_lines", ADD COLUMN "manual_usage_based_line_config_id" character(26) NULL, ADD COLUMN "parent_line_id" character(26) NULL, ADD
 CONSTRAINT "billing_invoice_lines_billing_invoices_billing_invoice_lines" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_lines_child_lines" FOREIGN KEY ("parent_line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_manual_line_configs_manua" FOREIGN KEY ("manual_line_config_id") REFERENCES "billing_invoice_manual_line_configs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_manual_usage_based_line_c" FOREIGN KEY ("manual_usage_based_line_config_id") REFERENCES "billing_invoice_manual_usage_based_line_configs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- create index "billinginvoiceline_namespace_parent_line_id" to table: "billing_invoice_lines"
CREATE INDEX "billinginvoiceline_namespace_parent_line_id" ON "billing_invoice_lines" ("namespace", "parent_line_id");
-- modify "billing_invoice_validation_issues" table
ALTER TABLE "billing_invoice_validation_issues" DROP CONSTRAINT "billing_invoice_validation_issues_billing_invoices_billing_invo", ADD
 CONSTRAINT "billing_invoice_validation_issues_billing_invoices_billing_invo" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
