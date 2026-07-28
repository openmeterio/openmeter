-- reverse: create index "plan_custom_currency_id" to table: "plans"
DROP INDEX "plan_custom_currency_id";
-- reverse: modify "plans" table
ALTER TABLE "plans" DROP CONSTRAINT "plans_custom_currencies_plans", DROP COLUMN "custom_currency_id", ALTER COLUMN "currency" SET DEFAULT 'USD';
-- reverse: create index "planratecard_custom_currency_id" to table: "plan_rate_cards"
DROP INDEX "planratecard_custom_currency_id";
-- reverse: modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP CONSTRAINT "plan_rate_cards_custom_currencies_plan_rate_cards", DROP COLUMN "custom_currency_id", DROP COLUMN "currency";
-- reverse: create index "addon_custom_currency_id" to table: "addons"
DROP INDEX "addon_custom_currency_id";
-- reverse: modify "addons" table
ALTER TABLE "addons" DROP CONSTRAINT "addons_custom_currencies_addons", DROP COLUMN "custom_currency_id", ALTER COLUMN "currency" SET DEFAULT 'USD';
-- reverse: create index "addonratecard_custom_currency_id" to table: "addon_rate_cards"
DROP INDEX "addonratecard_custom_currency_id";
-- reverse: modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP CONSTRAINT "addon_rate_cards_custom_currencies_addon_rate_cards", DROP COLUMN "custom_currency_id", DROP COLUMN "currency";
