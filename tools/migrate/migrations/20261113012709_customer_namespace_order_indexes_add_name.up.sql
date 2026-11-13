-- create index "customer_namespace_name" to table: "customers"
CREATE INDEX CONCURRENTLY "customer_namespace_name" ON "customers" ("namespace", "name");
