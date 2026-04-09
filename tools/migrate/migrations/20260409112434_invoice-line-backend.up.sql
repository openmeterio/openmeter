-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "engine" character varying NOT NULL DEFAULT 'invoicing';
