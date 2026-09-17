-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_item_tax_code_consistency", DROP CONSTRAINT "subscription_item_tax_behavior_consistency";
