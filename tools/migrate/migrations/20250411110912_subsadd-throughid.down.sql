-- reverse: create index "subscriptionaddonratecarditemlink_subscription_item_through_id" to table: "subscription_addon_rate_card_item_links"
DROP INDEX "subscriptionaddonratecarditemlink_subscription_item_through_id";
-- reverse: modify "subscription_addon_rate_card_item_links" table
ALTER TABLE "subscription_addon_rate_card_item_links" DROP COLUMN "subscription_item_through_id";
