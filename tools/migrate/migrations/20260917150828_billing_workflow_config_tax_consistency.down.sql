-- reverse: modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" DROP CONSTRAINT "billing_workflow_config_tax_code_consistency", DROP CONSTRAINT "billing_workflow_config_tax_behavior_consistency";
