-- reverse: modify "customers" table
ALTER TABLE "customers" ADD COLUMN "timezone" character varying NULL;
-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "customer_timezone" character varying NULL;
