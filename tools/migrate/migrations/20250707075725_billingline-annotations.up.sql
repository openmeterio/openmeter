-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "annotations" jsonb NULL;
-- create index "billinginvoiceline_annotations" to table: "billing_invoice_lines"
CREATE INDEX "billinginvoiceline_annotations" ON "billing_invoice_lines" USING gin ("annotations");
