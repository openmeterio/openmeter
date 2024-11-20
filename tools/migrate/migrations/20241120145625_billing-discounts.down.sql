-- reverse: create index "billinginvoicelinediscount_namespace_line_id_child_unique_refer" to table: "billing_invoice_line_discounts"
DROP INDEX "billinginvoicelinediscount_namespace_line_id_child_unique_refer";
-- reverse: create index "billinginvoicelinediscount_namespace_line_id" to table: "billing_invoice_line_discounts"
DROP INDEX "billinginvoicelinediscount_namespace_line_id";
-- reverse: create index "billinginvoicelinediscount_namespace" to table: "billing_invoice_line_discounts"
DROP INDEX "billinginvoicelinediscount_namespace";
-- reverse: create index "billinginvoicelinediscount_id" to table: "billing_invoice_line_discounts"
DROP INDEX "billinginvoicelinediscount_id";
-- reverse: create "billing_invoice_line_discounts" table
DROP TABLE "billing_invoice_line_discounts";
-- reverse: create index "billinginvoiceline_namespace_parent_line_id_child_unique_refere" to table: "billing_invoice_lines"
DROP INDEX "billinginvoiceline_namespace_parent_line_id_child_unique_refere";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_billing_invoice_lines_detailed_lines", DROP COLUMN "child_unique_reference_id", ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_lines_child_lines" FOREIGN KEY ("parent_line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- reverse: rename a column from "amount" to "per_unit_amount"
ALTER TABLE "billing_invoice_flat_fee_line_configs" RENAME COLUMN "per_unit_amount" TO "amount";
