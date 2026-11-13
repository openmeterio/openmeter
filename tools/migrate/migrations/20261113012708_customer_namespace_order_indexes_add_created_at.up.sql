-- create index "customer_namespace_created_at" to table: "customers"
CREATE INDEX CONCURRENTLY "customer_namespace_created_at" ON "customers" ("namespace", "created_at");
