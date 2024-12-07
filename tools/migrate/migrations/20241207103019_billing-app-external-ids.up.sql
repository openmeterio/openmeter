-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "invoicing_app_external_id" character varying NULL;
-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "invoicing_app_external_id" character varying NULL, ADD COLUMN "payment_app_external_id" character varying NULL;
