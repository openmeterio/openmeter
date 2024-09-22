-- reverse: create index "billingcustomeroverride_namespace_id" to table: "billing_customer_overrides"
DROP INDEX "billingcustomeroverride_namespace_id";
-- reverse: create index "billingcustomeroverride_namespace_customer_id" to table: "billing_customer_overrides"
DROP INDEX "billingcustomeroverride_namespace_customer_id";
-- reverse: create index "billingcustomeroverride_namespace" to table: "billing_customer_overrides"
DROP INDEX "billingcustomeroverride_namespace";
-- reverse: create index "billingcustomeroverride_id" to table: "billing_customer_overrides"
DROP INDEX "billingcustomeroverride_id";
-- reverse: create index "billing_customer_overrides_customer_id_key" to table: "billing_customer_overrides"
DROP INDEX "billing_customer_overrides_customer_id_key";
-- reverse: create "billing_customer_overrides" table
DROP TABLE "billing_customer_overrides";
