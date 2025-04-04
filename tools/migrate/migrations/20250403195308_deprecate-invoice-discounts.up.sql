-- modify "billing_invoice_discounts" table
ALTER TABLE "billing_invoice_discounts" DROP CONSTRAINT "billing_invoice_discounts_billing_invoices_invoice_discounts";
-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_billing_invoice_discounts_lines";
