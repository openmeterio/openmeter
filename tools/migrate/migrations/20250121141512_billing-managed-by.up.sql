-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "managed_by" character varying;
UPDATE "billing_invoice_lines" SET "managed_by" = 'system' WHERE "status" = 'detailed';
UPDATE "billing_invoice_lines" SET "managed_by" = 'subscription' WHERE "subscription_id" IS NOT NULL and "status" <> 'detailed';
UPDATE "billing_invoice_lines" SET "managed_by" = 'manual' WHERE "subscription_id" IS NULL and "status" <> 'detailed';
-- atlas:nolint MF104
ALTER TABLE "billing_invoice_lines" ALTER COLUMN "managed_by" SET NOT NULL;
