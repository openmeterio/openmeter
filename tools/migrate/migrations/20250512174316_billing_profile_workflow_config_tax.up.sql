-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" ADD COLUMN "tax_enabled" boolean NOT NULL DEFAULT true, ADD COLUMN "tax_enforced" boolean NOT NULL DEFAULT false;
