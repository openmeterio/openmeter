-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD COLUMN "unit_config" jsonb NULL;
-- modify "billing_invoice_usage_based_line_configs" table
ALTER TABLE "billing_invoice_usage_based_line_configs" ADD COLUMN "unit_config" jsonb NULL;
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD COLUMN "unit_config" jsonb NULL;
-- modify "subscription_items" table
ALTER TABLE "subscription_items" ADD COLUMN "unit_config" jsonb NULL;
-- modify "features" table
ALTER TABLE "features" ADD COLUMN "unit_config" jsonb NULL;
