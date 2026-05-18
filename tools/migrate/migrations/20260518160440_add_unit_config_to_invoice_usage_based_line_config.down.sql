-- reverse: modify "billing_invoice_usage_based_line_configs" table
ALTER TABLE "billing_invoice_usage_based_line_configs" DROP COLUMN "applied_unit_config", DROP COLUMN "converted_quantity";
