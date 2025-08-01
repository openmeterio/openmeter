-- reverse: create index "billinginvoicedetailedline_namespace_parent_line_id_child_uniqu" to table: "billing_invoice_detailed_lines"
DROP INDEX "billinginvoicedetailedline_namespace_parent_line_id_child_uniqu";
-- reverse: create index "billinginvoicedetailedline_namespace_parent_line_id" to table: "billing_invoice_detailed_lines"
DROP INDEX "billinginvoicedetailedline_namespace_parent_line_id";
-- reverse: create index "billinginvoicedetailedline_namespace_invoice_id" to table: "billing_invoice_detailed_lines"
DROP INDEX "billinginvoicedetailedline_namespace_invoice_id";
-- reverse: create index "billinginvoicedetailedline_namespace_id" to table: "billing_invoice_detailed_lines"
DROP INDEX "billinginvoicedetailedline_namespace_id";
-- reverse: create index "billinginvoicedetailedline_namespace" to table: "billing_invoice_detailed_lines"
DROP INDEX "billinginvoicedetailedline_namespace";
-- reverse: create index "billinginvoicedetailedline_id" to table: "billing_invoice_detailed_lines"
DROP INDEX "billinginvoicedetailedline_id";
-- reverse: create index "billinginvoicedetailedline_annotations" to table: "billing_invoice_detailed_lines"
DROP INDEX "billinginvoicedetailedline_annotations";
-- reverse: create "billing_invoice_detailed_lines" table
DROP TABLE "billing_invoice_detailed_lines";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "fee_line_config_id" character(26) NULL, ADD CONSTRAINT "billing_invoice_lines_billing_invoice_lines_detailed_lines" FOREIGN KEY ("parent_line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
