-- modify "billing_customer_overrides" table
ALTER TABLE "billing_customer_overrides" ADD COLUMN "invoice_progressive_billing" boolean NULL;
-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" ADD COLUMN "invoice_progressive_billing" boolean NOT NULL DEFAULT FALSE;
ALTER TABLE "billing_workflow_configs" ALTER COLUMN "invoice_progressive_billing" DROP DEFAULT;
