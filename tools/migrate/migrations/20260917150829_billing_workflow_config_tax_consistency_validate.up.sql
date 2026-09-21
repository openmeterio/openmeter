-- Validate the billing-workflow-config tax consistency checks separately from the
-- data repair so PostgreSQL does not hold validation locks for its duration.
ALTER TABLE "billing_workflow_configs"
  VALIDATE CONSTRAINT "billing_workflow_config_tax_behavior_consistency";

ALTER TABLE "billing_workflow_configs"
  VALIDATE CONSTRAINT "billing_workflow_config_tax_code_consistency";
