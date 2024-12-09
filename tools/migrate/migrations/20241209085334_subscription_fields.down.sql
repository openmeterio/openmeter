-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_items_entitlements_subscription_item", ADD
 CONSTRAINT "subscription_items_entitlements_entitlement" FOREIGN KEY ("entitlement_id") REFERENCES "entitlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- reverse: modify "subscriptions" table
ALTER TABLE "subscriptions" DROP COLUMN "description", DROP COLUMN "name";
-- reverse: modify "entitlements" table
ALTER TABLE "entitlements" ADD COLUMN "entitlement_subscription_item" character(26) NULL;
-- reverse: custom
ALTER TABLE "entitlements" ADD
 CONSTRAINT "entitlements_subscription_items_subscription_item" FOREIGN KEY ("entitlement_subscription_item") REFERENCES "subscription_items" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
