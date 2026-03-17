-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "lifecycle_handler" character varying NOT NULL DEFAULT 'default';
