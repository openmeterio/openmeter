-- reverse: create index "billinginvoiceline_annotations" to table: "billing_invoice_lines"
DROP INDEX "billinginvoiceline_annotations";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP COLUMN "annotations";
