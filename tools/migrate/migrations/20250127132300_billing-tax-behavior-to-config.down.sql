-- reverse: modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" DROP COLUMN "invoice_default_tax_settings", ADD COLUMN "invoice_tax_behavior" character varying NULL;
-- reverse: modify "billing_customer_overrides" table
ALTER TABLE "billing_customer_overrides" DROP COLUMN "invoice_default_tax_config", ADD COLUMN "invoice_tax_behavior" character varying NULL;
