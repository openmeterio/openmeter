-- modify "subscription_items" table
ALTER TABLE "subscription_items" ADD CONSTRAINT "subscription_item_currency_has_price" CHECK ((price IS NOT NULL) OR ((currency IS NULL) AND (custom_currency_id IS NULL))), ADD CONSTRAINT "subscription_item_currency_reference" CHECK ((currency IS NULL) OR (custom_currency_id IS NULL));
