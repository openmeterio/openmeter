-- modify "billing_invoice_items" table
-- atlas:nolint MF103
ALTER TABLE "billing_invoice_items" ALTER COLUMN "quantity" DROP NOT NULL, ADD COLUMN "type" character varying NOT NULL, ADD COLUMN "name" character varying NOT NULL;
-- modify "billing_invoices" table
-- atlas:nolint DS103
ALTER TABLE "billing_invoices" DROP COLUMN "key", DROP COLUMN "total_amount", ADD COLUMN "series" character varying NULL, ADD COLUMN "code" character varying NULL;
