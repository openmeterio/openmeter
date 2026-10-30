-- create index "customer_namespace_updated_at" to table: "customers"
CREATE INDEX "customer_namespace_updated_at" ON "customers" ("namespace", "updated_at");
