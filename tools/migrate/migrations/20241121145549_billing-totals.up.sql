-- modify "billing_invoice_flat_fee_line_configs" table
ALTER TABLE "billing_invoice_flat_fee_line_configs" ADD COLUMN "category" character varying NOT NULL DEFAULT 'regular';
-- modify "billing_invoice_lines" table
-- atlas:nolint MF103
ALTER TABLE "billing_invoice_lines" ADD COLUMN "amount" numeric NOT NULL, ADD COLUMN "taxes_total" numeric NOT NULL, ADD COLUMN "taxes_inclusive_total" numeric NOT NULL, ADD COLUMN "taxes_exclusive_total" numeric NOT NULL, ADD COLUMN "charges_total" numeric NOT NULL, ADD COLUMN "discounts_total" numeric NOT NULL, ADD COLUMN "total" numeric NOT NULL;
-- modify "billing_invoices" table
-- atlas:nolint MF103
ALTER TABLE "billing_invoices" ADD COLUMN "amount" numeric NOT NULL, ADD COLUMN "taxes_total" numeric NOT NULL, ADD COLUMN "taxes_inclusive_total" numeric NOT NULL, ADD COLUMN "taxes_exclusive_total" numeric NOT NULL, ADD COLUMN "charges_total" numeric NOT NULL, ADD COLUMN "discounts_total" numeric NOT NULL, ADD COLUMN "total" numeric NOT NULL;
