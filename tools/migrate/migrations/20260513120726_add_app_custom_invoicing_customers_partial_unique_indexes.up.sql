-- drop index "appcustominvoicingcustomer_namespace_app_id_customer_id" from table: "app_custom_invoicing_customers"
DROP INDEX "appcustominvoicingcustomer_namespace_app_id_customer_id";
-- create index "appcustominvoicingcustomer_namespace_app_id_customer_id" to table: "app_custom_invoicing_customers"
CREATE UNIQUE INDEX "appcustominvoicingcustomer_namespace_app_id_customer_id" ON "app_custom_invoicing_customers" ("namespace", "app_id", "customer_id") WHERE (deleted_at IS NULL);
