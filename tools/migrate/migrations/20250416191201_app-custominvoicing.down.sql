-- reverse: create index "appcustominvoicingcustomer_namespace_app_id_customer_id" to table: "app_custom_invoicing_customers"
DROP INDEX "appcustominvoicingcustomer_namespace_app_id_customer_id";
-- reverse: create index "appcustominvoicingcustomer_namespace" to table: "app_custom_invoicing_customers"
DROP INDEX "appcustominvoicingcustomer_namespace";
-- reverse: create "app_custom_invoicing_customers" table
DROP TABLE "app_custom_invoicing_customers";
-- reverse: create index "appcustominvoicing_namespace" to table: "app_custom_invoicings"
DROP INDEX "appcustominvoicing_namespace";
-- reverse: create index "appcustominvoicing_id" to table: "app_custom_invoicings"
DROP INDEX "appcustominvoicing_id";
-- reverse: create "app_custom_invoicings" table
DROP TABLE "app_custom_invoicings";
