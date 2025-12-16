-- reverse: create index "subscriptionbillingsyncstate_namespace_subscription_id" to table: "subscription_billing_sync_states"
DROP INDEX "subscriptionbillingsyncstate_namespace_subscription_id";
-- reverse: create index "subscriptionbillingsyncstate_namespace" to table: "subscription_billing_sync_states"
DROP INDEX "subscriptionbillingsyncstate_namespace";
-- reverse: create index "subscriptionbillingsyncstate_id" to table: "subscription_billing_sync_states"
DROP INDEX "subscriptionbillingsyncstate_id";
-- reverse: create index "subscription_billing_sync_states_subscription_id_key" to table: "subscription_billing_sync_states"
DROP INDEX "subscription_billing_sync_states_subscription_id_key";
-- reverse: create "subscription_billing_sync_states" table
DROP TABLE "subscription_billing_sync_states";
