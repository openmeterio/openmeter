-- modify "billing_invoice_usage_based_line_configs" table
ALTER TABLE "billing_invoice_usage_based_line_configs" ALTER COLUMN "feature_key" DROP NOT NULL;
