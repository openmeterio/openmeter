-- modify "entitlements" table
ALTER TABLE "entitlements" DROP COLUMN "entitlement_subscription_item";
-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "name" character varying NOT NULL DEFAULT 'Subscription', ADD COLUMN "description" character varying NULL;
-- modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_items_entitlements_entitlement", ADD
 CONSTRAINT "subscription_items_entitlements_subscription_item" FOREIGN KEY ("entitlement_id") REFERENCES "entitlements" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
