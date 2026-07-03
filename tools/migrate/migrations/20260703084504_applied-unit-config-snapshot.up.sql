-- modify "billing_invoice_usage_based_line_configs" table
ALTER TABLE "billing_invoice_usage_based_line_configs" ADD COLUMN "applied_unit_config" jsonb NULL;
