-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_line_tax_behavior_consistency", DROP CONSTRAINT "billing_invoice_line_tax_code_consistency", ADD COLUMN "invoicing_app_tax_code" character varying NULL;
