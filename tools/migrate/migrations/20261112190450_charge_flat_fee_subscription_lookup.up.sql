-- create index "chargeflatfee_namespace_subscription_id" to table: "charge_flat_fees"
CREATE INDEX CONCURRENTLY "chargeflatfee_namespace_subscription_id" ON "charge_flat_fees" ("namespace", "subscription_id");
