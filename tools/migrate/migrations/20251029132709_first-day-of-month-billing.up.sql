-- modify "billing_customer_overrides" table
ALTER TABLE "billing_customer_overrides" ADD COLUMN "anchored_alignment_detail" jsonb NULL;
-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" ADD COLUMN "anchored_alignment_detail" jsonb NULL;
