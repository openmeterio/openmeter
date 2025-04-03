-- reverse: create index "subscriptionaddonratecarditemlink_subscription_item_id_subscrip" to table: "subscription_addon_rate_card_item_links"
DROP INDEX "subscriptionaddonratecarditemlink_subscription_item_id_subscrip";
-- reverse: create index "subscriptionaddonratecarditemlink_subscription_item_id" to table: "subscription_addon_rate_card_item_links"
DROP INDEX "subscriptionaddonratecarditemlink_subscription_item_id";
-- reverse: create index "subscriptionaddonratecarditemlink_subscription_addon_rate_card_" to table: "subscription_addon_rate_card_item_links"
DROP INDEX "subscriptionaddonratecarditemlink_subscription_addon_rate_card_";
-- reverse: create index "subscriptionaddonratecarditemlink_id" to table: "subscription_addon_rate_card_item_links"
DROP INDEX "subscriptionaddonratecarditemlink_id";
-- reverse: create "subscription_addon_rate_card_item_links" table
DROP TABLE "subscription_addon_rate_card_item_links";
-- reverse: create index "subscriptionaddonratecard_subscription_addon_id" to table: "subscription_addon_rate_cards"
DROP INDEX "subscriptionaddonratecard_subscription_addon_id";
-- reverse: create index "subscriptionaddonratecard_namespace_id" to table: "subscription_addon_rate_cards"
DROP INDEX "subscriptionaddonratecard_namespace_id";
-- reverse: create index "subscriptionaddonratecard_namespace" to table: "subscription_addon_rate_cards"
DROP INDEX "subscriptionaddonratecard_namespace";
-- reverse: create index "subscriptionaddonratecard_id" to table: "subscription_addon_rate_cards"
DROP INDEX "subscriptionaddonratecard_id";
-- reverse: create "subscription_addon_rate_cards" table
DROP TABLE "subscription_addon_rate_cards";
-- reverse: create index "subscriptionaddonquantity_subscription_addon_id" to table: "subscription_addon_quantities"
DROP INDEX "subscriptionaddonquantity_subscription_addon_id";
-- reverse: create index "subscriptionaddonquantity_namespace" to table: "subscription_addon_quantities"
DROP INDEX "subscriptionaddonquantity_namespace";
-- reverse: create index "subscriptionaddonquantity_id" to table: "subscription_addon_quantities"
DROP INDEX "subscriptionaddonquantity_id";
-- reverse: create "subscription_addon_quantities" table
DROP TABLE "subscription_addon_quantities";
-- reverse: create index "subscriptionaddon_namespace" to table: "subscription_addons"
DROP INDEX "subscriptionaddon_namespace";
-- reverse: create index "subscriptionaddon_id" to table: "subscription_addons"
DROP INDEX "subscriptionaddon_id";
-- reverse: create "subscription_addons" table
DROP TABLE "subscription_addons";
