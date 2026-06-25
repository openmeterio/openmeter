-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD COLUMN "unit_config" jsonb NULL;
-- modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ADD COLUMN "unit_config" jsonb NULL;
-- modify "charge_usage_based_overrides" table
ALTER TABLE "charge_usage_based_overrides" ADD COLUMN "unit_config" jsonb NULL;
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD COLUMN "unit_config" jsonb NULL;
