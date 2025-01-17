-- modify "billing_invoice_flat_fee_line_configs" table
ALTER TABLE "billing_invoice_flat_fee_line_configs" ADD COLUMN "payment_term" character varying NOT NULL DEFAULT 'in_advance';
