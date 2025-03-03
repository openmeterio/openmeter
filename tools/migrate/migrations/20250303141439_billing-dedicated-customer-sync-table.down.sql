-- reverse: create index "billingcustomerlock_namespace_customer_id" to table: "billing_customer_locks"
DROP INDEX "billingcustomerlock_namespace_customer_id";
-- reverse: create index "billingcustomerlock_namespace" to table: "billing_customer_locks"
DROP INDEX "billingcustomerlock_namespace";
-- reverse: create index "billingcustomerlock_id" to table: "billing_customer_locks"
DROP INDEX "billingcustomerlock_id";
-- reverse: create "billing_customer_locks" table
DROP TABLE "billing_customer_locks";
