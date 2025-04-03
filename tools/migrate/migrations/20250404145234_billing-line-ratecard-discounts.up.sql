-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "ratecard_discounts" jsonb NULL;
