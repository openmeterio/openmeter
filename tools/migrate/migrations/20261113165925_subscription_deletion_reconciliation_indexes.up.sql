-- drop index "subscriptionsyncstatev2_namespace_subscription_id" from table: "subscription_sync_states_v2"
DROP INDEX "subscriptionsyncstatev2_namespace_subscription_id";
-- create index "subscriptionsyncstatev2_namespace_subscription_id" to table: "subscription_sync_states_v2"
CREATE UNIQUE INDEX "subscriptionsyncstatev2_namespace_subscription_id" ON "subscription_sync_states_v2" ("namespace", "subscription_id") INCLUDE ("last_success_state");
-- create index "subscription_deleted_reconciliation_cursor" to table: "subscriptions"
CREATE INDEX "subscription_deleted_reconciliation_cursor" ON "subscriptions" ("namespace", "created_at", "id") WHERE (deleted_at IS NOT NULL);
