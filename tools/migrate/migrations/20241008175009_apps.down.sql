-- reverse: create index "appstripecustomer_namespace_app_id_customer_id" to table: "app_stripe_customers"
DROP INDEX "appstripecustomer_namespace_app_id_customer_id";
-- reverse: create index "appstripecustomer_namespace" to table: "app_stripe_customers"
DROP INDEX "appstripecustomer_namespace";
-- reverse: create index "appstripecustomer_app_id_stripe_customer_id" to table: "app_stripe_customers"
DROP INDEX "appstripecustomer_app_id_stripe_customer_id";
-- reverse: create "app_stripe_customers" table
DROP TABLE "app_stripe_customers";
-- reverse: create index "appstripe_namespace" to table: "app_stripes"
DROP INDEX "appstripe_namespace";
-- reverse: create index "appstripe_id" to table: "app_stripes"
DROP INDEX "appstripe_id";
-- reverse: create "app_stripes" table
DROP TABLE "app_stripes";
-- reverse: create index "appcustomer_namespace_app_id_customer_id" to table: "app_customers"
DROP INDEX "appcustomer_namespace_app_id_customer_id";
-- reverse: create index "appcustomer_namespace" to table: "app_customers"
DROP INDEX "appcustomer_namespace";
-- reverse: create "app_customers" table
DROP TABLE "app_customers";
-- reverse: modify "customers" table
ALTER TABLE "customers" ADD COLUMN "external_mapping_stripe_customer_id" character varying NULL;
-- reverse: create index "app_namespace_type_is_default" to table: "apps"
DROP INDEX "app_namespace_type_is_default";
-- reverse: create index "app_namespace_type" to table: "apps"
DROP INDEX "app_namespace_type";
-- reverse: create index "app_namespace_id" to table: "apps"
DROP INDEX "app_namespace_id";
-- reverse: create index "app_namespace" to table: "apps"
DROP INDEX "app_namespace";
-- reverse: create index "app_id" to table: "apps"
DROP INDEX "app_id";
-- reverse: create "apps" table
DROP TABLE "apps";
