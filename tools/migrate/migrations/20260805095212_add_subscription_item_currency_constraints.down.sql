-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_item_currency_reference", DROP CONSTRAINT "subscription_item_currency_has_price";
