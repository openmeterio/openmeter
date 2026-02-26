-- reverse: modify "standard_invoice_settlements" table
ALTER TABLE "standard_invoice_settlements" DROP COLUMN "credits_total";
-- reverse: modify "billing_standard_invoice_detailed_lines" table
ALTER TABLE "billing_standard_invoice_detailed_lines" DROP COLUMN "credits_applied", DROP COLUMN "credits_total";
-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" DROP COLUMN "credits_total";
-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP COLUMN "credits_applied", DROP COLUMN "credits_total";
