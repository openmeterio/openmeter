-- reverse: create index "subscriptionitem_custom_currency_id" to table: "subscription_items"
DROP INDEX "subscriptionitem_custom_currency_id";
-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_items_custom_currencies_subscription_items", DROP CONSTRAINT "subscription_item_currency_reference", DROP CONSTRAINT "subscription_item_currency_has_price", DROP COLUMN "custom_currency_id", DROP COLUMN "currency";
