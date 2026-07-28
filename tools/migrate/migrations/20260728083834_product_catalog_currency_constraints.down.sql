-- reverse: modify "plans" table
ALTER TABLE "plans" DROP CONSTRAINT "plan_currency_reference";
-- reverse: modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP CONSTRAINT "plan_rate_card_currency_reference", DROP CONSTRAINT "plan_rate_card_currency_has_price";
-- reverse: modify "addons" table
ALTER TABLE "addons" DROP CONSTRAINT "addon_currency_reference";
-- reverse: modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP CONSTRAINT "addon_rate_card_currency_reference", DROP CONSTRAINT "addon_rate_card_currency_has_price";
