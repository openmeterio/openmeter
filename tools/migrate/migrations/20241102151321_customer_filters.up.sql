-- create index "customer_created_at" to table: "customers"
CREATE INDEX "customer_created_at" ON "customers" ("created_at");
-- create index "customer_deleted_at" to table: "customers"
CREATE INDEX "customer_deleted_at" ON "customers" ("deleted_at");
-- create index "customer_name" to table: "customers"
CREATE INDEX "customer_name" ON "customers" ("name");
-- create index "customer_primary_email" to table: "customers"
CREATE INDEX "customer_primary_email" ON "customers" ("primary_email");
