-- reverse: modify "billing_invoice_validation_issues" table
ALTER TABLE "billing_invoice_validation_issues" DROP CONSTRAINT "billing_invoice_validation_issues_billing_invoices_billing_invo", ADD
 CONSTRAINT "billing_invoice_validation_issues_billing_invoices_billing_invo" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- reverse: create index "billinginvoiceline_namespace_parent_line_id" to table: "billing_invoice_lines"
DROP INDEX "billinginvoiceline_namespace_parent_line_id";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_billing_invoice_manual_usage_based_line_c", DROP CONSTRAINT "billing_invoice_lines_billing_invoice_manual_line_configs_manua", DROP CONSTRAINT "billing_invoice_lines_billing_invoice_lines_child_lines", DROP CONSTRAINT "billing_invoice_lines_billing_invoices_billing_invoice_lines", DROP COLUMN "parent_line_id", DROP COLUMN "manual_usage_based_line_config_id", ADD
 CONSTRAINT "billing_invoice_lines_billing_invoices_billing_invoice_lines" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_manual_line_configs_billi" FOREIGN KEY ("manual_line_config_id") REFERENCES "billing_invoice_manual_line_configs" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- reverse: create index "billinginvoicemanualusagebasedlineconfig_namespace" to table: "billing_invoice_manual_usage_based_line_configs"
DROP INDEX "billinginvoicemanualusagebasedlineconfig_namespace";
-- reverse: create index "billinginvoicemanualusagebasedlineconfig_id" to table: "billing_invoice_manual_usage_based_line_configs"
DROP INDEX "billinginvoicemanualusagebasedlineconfig_id";
-- reverse: create "billing_invoice_manual_usage_based_line_configs" table
DROP TABLE "billing_invoice_manual_usage_based_line_configs";
-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" DROP COLUMN "customer_usage_attribution";
