-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" DROP COLUMN "total", DROP COLUMN "discounts_total", DROP COLUMN "charges_total", DROP COLUMN "taxes_exclusive_total", DROP COLUMN "taxes_inclusive_total", DROP COLUMN "taxes_total", DROP COLUMN "amount";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP COLUMN "total", DROP COLUMN "discounts_total", DROP COLUMN "charges_total", DROP COLUMN "taxes_exclusive_total", DROP COLUMN "taxes_inclusive_total", DROP COLUMN "taxes_total", DROP COLUMN "amount";
-- reverse: modify "billing_invoice_flat_fee_line_configs" table
ALTER TABLE "billing_invoice_flat_fee_line_configs" DROP COLUMN "category";
