-- reverse: modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" DROP COLUMN "tax_enforced", DROP COLUMN "tax_enabled";
