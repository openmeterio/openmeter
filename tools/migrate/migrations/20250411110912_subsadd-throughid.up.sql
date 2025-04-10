-- modify "subscription_addon_rate_card_item_links" table
ALTER TABLE "subscription_addon_rate_card_item_links" ADD COLUMN "subscription_item_through_id" character varying NULL;
-- create index "subscriptionaddonratecarditemlink_subscription_item_through_id" to table: "subscription_addon_rate_card_item_links"
CREATE INDEX "subscriptionaddonratecarditemlink_subscription_item_through_id" ON "subscription_addon_rate_card_item_links" ("subscription_item_through_id");
