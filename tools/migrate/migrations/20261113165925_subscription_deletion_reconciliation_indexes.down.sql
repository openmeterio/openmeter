-- reverse: create index "subscription_deleted_reconciliation_cursor" to table: "subscriptions"
DROP INDEX "subscription_deleted_reconciliation_cursor";
-- reverse: create index "subscriptionsyncstatev2_namespace_subscription_id" to table: "subscription_sync_states_v2"
DROP INDEX "subscriptionsyncstatev2_namespace_subscription_id";
-- reverse: drop index "subscriptionsyncstatev2_namespace_subscription_id" from table: "subscription_sync_states_v2"
CREATE UNIQUE INDEX "subscriptionsyncstatev2_namespace_subscription_id" ON "subscription_sync_states_v2" ("namespace", "subscription_id");
