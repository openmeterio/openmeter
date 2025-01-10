-- modify "billing_invoices" table
-- atlas:nolint DS103
ALTER TABLE "billing_invoices" DROP COLUMN "customer_timezone";
-- modify "customers" table
-- atlas:nolint DS103
ALTER TABLE "customers" DROP COLUMN "timezone";
