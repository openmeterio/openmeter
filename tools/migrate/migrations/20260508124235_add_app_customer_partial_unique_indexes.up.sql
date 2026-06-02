-- drop index "appcustomer_namespace_app_id_customer_id" from table: "app_customers"
DROP INDEX "appcustomer_namespace_app_id_customer_id";
-- create index "appcustomer_namespace_app_id_customer_id" to table: "app_customers"
CREATE UNIQUE INDEX "appcustomer_namespace_app_id_customer_id" ON "app_customers" ("namespace", "app_id", "customer_id") WHERE (deleted_at IS NULL);
-- drop index "appstripecustomer_app_id_stripe_customer_id" from table: "app_stripe_customers"
DROP INDEX "appstripecustomer_app_id_stripe_customer_id";
-- drop index "appstripecustomer_namespace_app_id_customer_id" from table: "app_stripe_customers"
DROP INDEX "appstripecustomer_namespace_app_id_customer_id";
-- create index "appstripecustomer_app_id_stripe_customer_id" to table: "app_stripe_customers"
CREATE UNIQUE INDEX "appstripecustomer_app_id_stripe_customer_id" ON "app_stripe_customers" ("app_id", "stripe_customer_id") WHERE (deleted_at IS NULL);
-- create index "appstripecustomer_namespace_app_id_customer_id" to table: "app_stripe_customers"
CREATE UNIQUE INDEX "appstripecustomer_namespace_app_id_customer_id" ON "app_stripe_customers" ("namespace", "app_id", "customer_id") WHERE (deleted_at IS NULL);
