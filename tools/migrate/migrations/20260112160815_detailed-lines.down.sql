-- reverse: create index "billingstdinvdetailedlineamntdiscount_ns_parent_child_id" to table: "billing_standard_invoice_detailed_line_amount_discounts"
DROP INDEX "billingstdinvdetailedlineamntdiscount_ns_parent_child_id";
-- reverse: create index "billingstandardinvoicedetailedlineamountdiscount_namespace_line" to table: "billing_standard_invoice_detailed_line_amount_discounts"
DROP INDEX "billingstandardinvoicedetailedlineamountdiscount_namespace_line";
-- reverse: create index "billingstandardinvoicedetailedlineamountdiscount_namespace" to table: "billing_standard_invoice_detailed_line_amount_discounts"
DROP INDEX "billingstandardinvoicedetailedlineamountdiscount_namespace";
-- reverse: create index "billingstandardinvoicedetailedlineamountdiscount_id" to table: "billing_standard_invoice_detailed_line_amount_discounts"
DROP INDEX "billingstandardinvoicedetailedlineamountdiscount_id";
-- reverse: create "billing_standard_invoice_detailed_line_amount_discounts" table
DROP TABLE "billing_standard_invoice_detailed_line_amount_discounts";
-- reverse: create index "billingstdinvdetailedline_ns_parent_child_id" to table: "billing_standard_invoice_detailed_lines"
DROP INDEX "billingstdinvdetailedline_ns_parent_child_id";
-- reverse: create index "billingstandardinvoicedetailedline_namespace_parent_line_id" to table: "billing_standard_invoice_detailed_lines"
DROP INDEX "billingstandardinvoicedetailedline_namespace_parent_line_id";
-- reverse: create index "billingstandardinvoicedetailedline_namespace_invoice_id" to table: "billing_standard_invoice_detailed_lines"
DROP INDEX "billingstandardinvoicedetailedline_namespace_invoice_id";
-- reverse: create index "billingstandardinvoicedetailedline_namespace_id" to table: "billing_standard_invoice_detailed_lines"
DROP INDEX "billingstandardinvoicedetailedline_namespace_id";
-- reverse: create index "billingstandardinvoicedetailedline_namespace" to table: "billing_standard_invoice_detailed_lines"
DROP INDEX "billingstandardinvoicedetailedline_namespace";
-- reverse: create index "billingstandardinvoicedetailedline_id" to table: "billing_standard_invoice_detailed_lines"
DROP INDEX "billingstandardinvoicedetailedline_id";
-- reverse: create index "billingstandardinvoicedetailedline_annotations" to table: "billing_standard_invoice_detailed_lines"
DROP INDEX "billingstandardinvoicedetailedline_annotations";
-- reverse: create "billing_standard_invoice_detailed_lines" table
DROP TABLE "billing_standard_invoice_detailed_lines";
-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" DROP COLUMN "schema_level";
-- reverse: create index "billinginvoicewriteschemalevel_id" to table: "billing_invoice_write_schema_levels"
DROP INDEX "billinginvoicewriteschemalevel_id";
-- reverse: create "billing_invoice_write_schema_levels" table
DROP TABLE "billing_invoice_write_schema_levels";
