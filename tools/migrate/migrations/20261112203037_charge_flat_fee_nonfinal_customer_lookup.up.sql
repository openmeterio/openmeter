-- create index "chargeflatfee_namespace_customer_id" to table: "charge_flat_fees"
CREATE INDEX CONCURRENTLY "chargeflatfee_namespace_customer_id" ON "charge_flat_fees" ("namespace", "customer_id") WHERE ((deleted_at IS NULL) AND ((status)::text <> 'final'::text));
