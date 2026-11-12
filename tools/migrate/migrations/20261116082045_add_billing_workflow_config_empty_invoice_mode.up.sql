-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" ADD COLUMN "empty_invoice_mode" character varying NOT NULL DEFAULT 'create';
