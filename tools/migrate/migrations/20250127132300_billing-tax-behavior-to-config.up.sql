-- modify "billing_customer_overrides" table
-- atlas:nolint DS103
ALTER TABLE "billing_customer_overrides" DROP COLUMN "invoice_tax_behavior", ADD COLUMN "invoice_default_tax_config" jsonb NULL;
-- modify "billing_workflow_configs" table
-- atlas:nolint DS103
ALTER TABLE "billing_workflow_configs" DROP COLUMN "invoice_tax_behavior", ADD COLUMN "invoice_default_tax_settings" jsonb NULL;
