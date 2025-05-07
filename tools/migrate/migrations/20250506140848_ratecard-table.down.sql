-- reverse: create index "subscription_items_ratecard_id_key" to table: "subscription_items"
DROP INDEX "subscription_items_ratecard_id_key";
-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_items_rate_cards_subscription_item", DROP COLUMN "ratecard_id";
-- reverse: create index "plan_rate_cards_ratecard_id_key" to table: "plan_rate_cards"
DROP INDEX "plan_rate_cards_ratecard_id_key";
-- reverse: modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP CONSTRAINT "plan_rate_cards_rate_cards_plan_ratecard", DROP COLUMN "ratecard_id";
-- reverse: create index "addon_rate_cards_ratecard_id_key" to table: "addon_rate_cards"
DROP INDEX "addon_rate_cards_ratecard_id_key";
-- reverse: modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP CONSTRAINT "addon_rate_cards_rate_cards_addon_ratecard", DROP COLUMN "ratecard_id";
-- reverse: create index "ratecard_namespace_key_deleted_at" to table: "rate_cards"
DROP INDEX "ratecard_namespace_key_deleted_at";
-- reverse: create index "ratecard_namespace_id" to table: "rate_cards"
DROP INDEX "ratecard_namespace_id";
-- reverse: create index "ratecard_namespace" to table: "rate_cards"
DROP INDEX "ratecard_namespace";
-- reverse: create index "ratecard_id" to table: "rate_cards"
DROP INDEX "ratecard_id";
-- reverse: create "rate_cards" table
DROP TABLE "rate_cards";
