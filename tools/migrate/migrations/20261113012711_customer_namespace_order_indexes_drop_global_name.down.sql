-- reverse: drop index "customer_name" from table: "customers"
CREATE INDEX CONCURRENTLY "customer_name" ON "customers" ("name");
