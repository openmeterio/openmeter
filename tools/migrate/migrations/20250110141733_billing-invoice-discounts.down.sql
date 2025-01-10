-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_billing_invoice_discounts_lines", DROP COLUMN "line_ids";
-- reverse: create index "billinginvoicediscount_namespace_invoice_id" to table: "billing_invoice_discounts"
DROP INDEX "billinginvoicediscount_namespace_invoice_id";
-- reverse: create index "billinginvoicediscount_namespace_id" to table: "billing_invoice_discounts"
DROP INDEX "billinginvoicediscount_namespace_id";
-- reverse: create index "billinginvoicediscount_namespace" to table: "billing_invoice_discounts"
DROP INDEX "billinginvoicediscount_namespace";
-- reverse: create index "billinginvoicediscount_id" to table: "billing_invoice_discounts"
DROP INDEX "billinginvoicediscount_id";
-- reverse: create "billing_invoice_discounts" table
DROP TABLE "billing_invoice_discounts";
