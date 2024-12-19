-- modify "billing_invoice_line_discounts" table
ALTER TABLE "billing_invoice_line_discounts" ADD COLUMN "invoicing_app_external_id" character varying NULL;
