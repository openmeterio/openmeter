-- modify "subscription_items" table
ALTER TABLE "subscription_items" ADD CONSTRAINT "subscription_item_currency_has_price" CHECK ((price IS NOT NULL) OR ((currency IS NULL) AND (custom_currency_id IS NULL))), ADD CONSTRAINT "subscription_item_currency_reference" CHECK ((currency IS NULL) OR (custom_currency_id IS NULL)), ADD COLUMN "currency" character varying NULL, ADD COLUMN "custom_currency_id" character(26) NULL, ADD CONSTRAINT "subscription_items_custom_currencies_subscription_items" FOREIGN KEY ("custom_currency_id") REFERENCES "custom_currencies" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT;
-- create index "subscriptionitem_custom_currency_id" to table: "subscription_items"
CREATE INDEX "subscriptionitem_custom_currency_id" ON "subscription_items" ("custom_currency_id");
