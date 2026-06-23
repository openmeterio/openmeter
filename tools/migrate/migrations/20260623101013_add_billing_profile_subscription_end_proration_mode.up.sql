-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" ADD COLUMN "subscription_end_proration_mode" character varying NOT NULL DEFAULT 'bill_full_period';
ALTER TABLE "billing_workflow_configs" ALTER COLUMN "subscription_end_proration_mode" SET DEFAULT 'bill_actual_period';
