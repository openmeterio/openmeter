-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" ALTER COLUMN "customer_usage_attribution" SET NOT NULL;
