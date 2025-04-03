-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_discounts_lines" FOREIGN KEY ("line_ids") REFERENCES "billing_invoice_discounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- reverse: modify "billing_invoice_discounts" table
ALTER TABLE "billing_invoice_discounts" ADD
 CONSTRAINT "billing_invoice_discounts_billing_invoices_invoice_discounts" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
