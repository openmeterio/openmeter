-- reverse: drop index "customer_created_at" from table: "customers"
CREATE INDEX CONCURRENTLY "customer_created_at" ON "customers" ("created_at");
