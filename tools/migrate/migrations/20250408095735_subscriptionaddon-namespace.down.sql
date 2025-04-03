-- reverse: modify "subscription_addons" table
ALTER TABLE "subscription_addons" DROP COLUMN "metadata";
-- reverse: create index "subscriptionaddonquantity_namespace" to table: "subscription_addon_quantities"
DROP INDEX "subscriptionaddonquantity_namespace";
-- reverse: modify "subscription_addon_quantities" table
ALTER TABLE "subscription_addon_quantities" DROP COLUMN "namespace";
