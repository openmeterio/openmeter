-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" DROP COLUMN "code", DROP COLUMN "series", ADD COLUMN "total_amount" numeric NOT NULL, ADD COLUMN "key" character varying NOT NULL;
-- reverse: modify "billing_invoice_items" table
ALTER TABLE "billing_invoice_items" DROP COLUMN "name", DROP COLUMN "type", ALTER COLUMN "quantity" SET NOT NULL;
