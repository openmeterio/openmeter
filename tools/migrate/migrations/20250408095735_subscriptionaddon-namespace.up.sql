-- modify "subscription_addon_quantities" table
ALTER TABLE "subscription_addon_quantities" ADD COLUMN "namespace" character varying NOT NULL;
-- create index "subscriptionaddonquantity_namespace" to table: "subscription_addon_quantities"
CREATE INDEX "subscriptionaddonquantity_namespace" ON "subscription_addon_quantities" ("namespace");
-- modify "subscription_addons" table
ALTER TABLE "subscription_addons" ADD COLUMN "metadata" jsonb NULL;
