-- create "billing_customer_locks" table
CREATE TABLE "billing_customer_locks" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "customer_id" character(26) NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "billingcustomerlock_id" to table: "billing_customer_locks"
CREATE UNIQUE INDEX "billingcustomerlock_id" ON "billing_customer_locks" ("id");
-- create index "billingcustomerlock_namespace" to table: "billing_customer_locks"
CREATE INDEX "billingcustomerlock_namespace" ON "billing_customer_locks" ("namespace");
-- create index "billingcustomerlock_namespace_customer_id" to table: "billing_customer_locks"
CREATE UNIQUE INDEX "billingcustomerlock_namespace_customer_id" ON "billing_customer_locks" ("namespace", "customer_id");
