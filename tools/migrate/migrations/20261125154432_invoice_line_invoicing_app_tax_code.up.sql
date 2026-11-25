-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "invoicing_app_tax_code" character varying NULL;
