-- reverse: create index "subscriptionsyncstatev2_namespace_subscription_id" to table: "subscription_sync_states_v2"
DROP INDEX "subscriptionsyncstatev2_namespace_subscription_id";
-- reverse: create index "subscriptionsyncstatev2_namespace" to table: "subscription_sync_states_v2"
DROP INDEX "subscriptionsyncstatev2_namespace";
-- reverse: create index "subscriptionsyncstatev2_id" to table: "subscription_sync_states_v2"
DROP INDEX "subscriptionsyncstatev2_id";
-- reverse: create index "subscription_sync_states_v2_subscription_id_key" to table: "subscription_sync_states_v2"
DROP INDEX "subscription_sync_states_v2_subscription_id_key";
-- reverse: create "subscription_sync_states_v2" table
DROP TABLE "subscription_sync_states_v2";
