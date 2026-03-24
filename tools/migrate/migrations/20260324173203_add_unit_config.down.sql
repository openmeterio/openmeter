-- reverse: modify "features" table
ALTER TABLE "features" DROP COLUMN "unit_config";
-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP COLUMN "unit_config";
-- reverse: modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP COLUMN "unit_config";
-- reverse: modify "billing_invoice_usage_based_line_configs" table
ALTER TABLE "billing_invoice_usage_based_line_configs" DROP COLUMN "unit_config";
-- reverse: modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP COLUMN "unit_config";
