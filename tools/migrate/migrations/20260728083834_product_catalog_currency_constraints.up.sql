-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD CONSTRAINT "addon_rate_card_currency_has_price" CHECK ((price IS NOT NULL) OR ((currency IS NULL) AND (custom_currency_id IS NULL))), ADD CONSTRAINT "addon_rate_card_currency_reference" CHECK ((currency IS NULL) OR (custom_currency_id IS NULL));
-- modify "addons" table
ALTER TABLE "addons" ADD CONSTRAINT "addon_currency_reference" CHECK ((currency IS NULL) <> (custom_currency_id IS NULL));
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD CONSTRAINT "plan_rate_card_currency_has_price" CHECK ((price IS NOT NULL) OR ((currency IS NULL) AND (custom_currency_id IS NULL))), ADD CONSTRAINT "plan_rate_card_currency_reference" CHECK ((currency IS NULL) OR (custom_currency_id IS NULL));
-- modify "plans" table
ALTER TABLE "plans" ADD CONSTRAINT "plan_currency_reference" CHECK ((currency IS NULL) <> (custom_currency_id IS NULL));
