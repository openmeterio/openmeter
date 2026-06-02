-- create index "subscriptionaddon_namespace_subscription_id_addon_id" to table: "subscription_addons"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "subscriptionaddon_namespace_subscription_id_addon_id" ON "subscription_addons" ("namespace", "subscription_id", "addon_id") WHERE (deleted_at IS NULL);
