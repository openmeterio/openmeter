-- create index "chargeusagebased_namespace_customer_id" to table: "charge_usage_based"
CREATE INDEX CONCURRENTLY "chargeusagebased_namespace_customer_id" ON "charge_usage_based" ("namespace", "customer_id") WHERE ((deleted_at IS NULL) AND ((status)::text <> 'final'::text));
