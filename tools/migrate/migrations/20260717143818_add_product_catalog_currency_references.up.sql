-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD CONSTRAINT "addon_rate_card_currency_has_price" CHECK ((price IS NOT NULL) OR ((currency IS NULL) AND (custom_currency_id IS NULL))), ADD CONSTRAINT "addon_rate_card_currency_reference" CHECK ((currency IS NULL) OR (custom_currency_id IS NULL)), ADD COLUMN "custom_currency_id" character(26) NULL, ADD CONSTRAINT "addon_rate_cards_custom_currencies_addon_rate_cards" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "addonratecard_custom_currency_id" to table: "addon_rate_cards"
CREATE INDEX "addonratecard_custom_currency_id" ON "addon_rate_cards" ("custom_currency_id");
-- modify "addons" table
ALTER TABLE "addons" ADD CONSTRAINT "addon_currency_reference" CHECK ((currency IS NULL) <> (custom_currency_id IS NULL)), ALTER COLUMN "currency" DROP NOT NULL, ALTER COLUMN "currency" DROP DEFAULT, ADD COLUMN "custom_currency_id" character(26) NULL, ADD CONSTRAINT "addons_custom_currencies_addons" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "addon_custom_currency_id" to table: "addons"
CREATE INDEX "addon_custom_currency_id" ON "addons" ("custom_currency_id");
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD CONSTRAINT "plan_rate_card_currency_has_price" CHECK ((price IS NOT NULL) OR ((currency IS NULL) AND (custom_currency_id IS NULL))), ADD CONSTRAINT "plan_rate_card_currency_reference" CHECK ((currency IS NULL) OR (custom_currency_id IS NULL)), ADD COLUMN "custom_currency_id" character(26) NULL, ADD CONSTRAINT "plan_rate_cards_custom_currencies_plan_rate_cards" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "planratecard_custom_currency_id" to table: "plan_rate_cards"
CREATE INDEX "planratecard_custom_currency_id" ON "plan_rate_cards" ("custom_currency_id");
-- modify "plans" table
ALTER TABLE "plans" ADD CONSTRAINT "plan_currency_reference" CHECK ((currency IS NULL) <> (custom_currency_id IS NULL)), ALTER COLUMN "currency" DROP NOT NULL, ALTER COLUMN "currency" DROP DEFAULT, ADD COLUMN "custom_currency_id" character(26) NULL, ADD CONSTRAINT "plans_custom_currencies_plans" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "plan_custom_currency_id" to table: "plans"
CREATE INDEX "plan_custom_currency_id" ON "plans" ("custom_currency_id");
