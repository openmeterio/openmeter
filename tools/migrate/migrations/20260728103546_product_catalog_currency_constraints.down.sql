-- reverse: modify "plans" table
ALTER TABLE "plans" DROP CONSTRAINT "plan_currency_reference", DROP CONSTRAINT "plan_currency_code_length";
-- reverse: modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP CONSTRAINT "plan_rate_card_currency_reference", DROP CONSTRAINT "plan_rate_card_currency_has_price", DROP CONSTRAINT "plan_rate_card_currency_code_length";
-- reverse: modify "addons" table
ALTER TABLE "addons" DROP CONSTRAINT "addon_currency_reference", DROP CONSTRAINT "addon_currency_code_length";
-- reverse: modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP CONSTRAINT "addon_rate_card_currency_reference", DROP CONSTRAINT "addon_rate_card_currency_has_price", DROP CONSTRAINT "addon_rate_card_currency_code_length";
