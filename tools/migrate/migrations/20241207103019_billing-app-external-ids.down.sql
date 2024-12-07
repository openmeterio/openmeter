-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" DROP COLUMN "payment_app_external_id", DROP COLUMN "invoicing_app_external_id";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP COLUMN "invoicing_app_external_id";
