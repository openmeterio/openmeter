-- create index "chargeusagebased_namespace_subscription_id" to table: "charge_usage_based"
CREATE INDEX CONCURRENTLY "chargeusagebased_namespace_subscription_id" ON "charge_usage_based" ("namespace", "subscription_id");
