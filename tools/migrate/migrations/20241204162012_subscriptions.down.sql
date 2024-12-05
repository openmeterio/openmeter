-- reverse: modify "subscriptions" table
ALTER TABLE "subscriptions" DROP CONSTRAINT "subscriptions_customers_subscription";
-- reverse: modify "subscription_phases" table
ALTER TABLE "subscription_phases" DROP CONSTRAINT "subscription_phases_subscriptions_phases";
-- reverse: modify "subscription_items" table
ALTER TABLE "subscription_items" DROP CONSTRAINT "subscription_items_subscription_phases_items", DROP CONSTRAINT "subscription_items_entitlements_entitlement";
-- reverse: modify "entitlements" table
ALTER TABLE "entitlements" DROP CONSTRAINT "entitlements_subscription_items_subscription_item";
-- reverse: create index "subscription_namespace_id" to table: "subscriptions"
DROP INDEX "subscription_namespace_id";
-- reverse: create index "subscription_namespace_customer_id" to table: "subscriptions"
DROP INDEX "subscription_namespace_customer_id";
-- reverse: create index "subscription_namespace" to table: "subscriptions"
DROP INDEX "subscription_namespace";
-- reverse: create index "subscription_id" to table: "subscriptions"
DROP INDEX "subscription_id";
-- reverse: create "subscriptions" table
DROP TABLE "subscriptions";
-- reverse: create index "subscriptionphase_namespace_subscription_id_key" to table: "subscription_phases"
DROP INDEX "subscriptionphase_namespace_subscription_id_key";
-- reverse: create index "subscriptionphase_namespace_subscription_id" to table: "subscription_phases"
DROP INDEX "subscriptionphase_namespace_subscription_id";
-- reverse: create index "subscriptionphase_namespace_id" to table: "subscription_phases"
DROP INDEX "subscriptionphase_namespace_id";
-- reverse: create index "subscriptionphase_namespace" to table: "subscription_phases"
DROP INDEX "subscriptionphase_namespace";
-- reverse: create index "subscriptionphase_id" to table: "subscription_phases"
DROP INDEX "subscriptionphase_id";
-- reverse: create "subscription_phases" table
DROP TABLE "subscription_phases";
-- reverse: create index "subscriptionitem_namespace_phase_id_key" to table: "subscription_items"
DROP INDEX "subscriptionitem_namespace_phase_id_key";
-- reverse: create index "subscriptionitem_namespace_id" to table: "subscription_items"
DROP INDEX "subscriptionitem_namespace_id";
-- reverse: create index "subscriptionitem_namespace" to table: "subscription_items"
DROP INDEX "subscriptionitem_namespace";
-- reverse: create index "subscriptionitem_id" to table: "subscription_items"
DROP INDEX "subscriptionitem_id";
-- reverse: create "subscription_items" table
DROP TABLE "subscription_items";
-- reverse: modify "entitlements" table
ALTER TABLE "entitlements" DROP COLUMN "entitlement_subscription_item", DROP COLUMN "subscription_managed";
