-- modify "billing_customer_overrides" table
ALTER TABLE "billing_customer_overrides" ADD COLUMN "invoice_tax_behavior" character varying NULL;
-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" ADD COLUMN "invoice_tax_behavior" character varying NULL;
