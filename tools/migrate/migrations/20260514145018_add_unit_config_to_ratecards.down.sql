-- reverse: modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP COLUMN "unit_config";
-- reverse: modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP COLUMN "unit_config";
