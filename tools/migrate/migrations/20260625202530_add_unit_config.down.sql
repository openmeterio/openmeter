-- reverse: modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP COLUMN "unit_config";
-- reverse: modify "charge_usage_based_overrides" table
ALTER TABLE "charge_usage_based_overrides" DROP COLUMN "unit_config";
-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP COLUMN "unit_config";
-- reverse: modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP COLUMN "unit_config";
