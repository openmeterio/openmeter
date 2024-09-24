-- modify "billing_invoices" table
-- atlas:nolint DS103
ALTER TABLE "billing_invoices" DROP COLUMN "provider_config", DROP COLUMN "provider_reference", ADD COLUMN "tax_provider" character varying NULL, ADD COLUMN "invoicing_provider" character varying NULL, ADD COLUMN "payment_provider" character varying NULL;
-- create index "billing_invoices_workflow_config_id_key" to table: "billing_invoices"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billing_invoices_workflow_config_id_key" ON "billing_invoices" ("workflow_config_id");
-- drop index "billingprofile_namespace_default" from table: "billing_profiles"
DROP INDEX "billingprofile_namespace_default";
-- drop index "billingprofile_namespace_id" from table: "billing_profiles"
DROP INDEX "billingprofile_namespace_id";
-- drop index "billingprofile_namespace_key" from table: "billing_profiles"
DROP INDEX "billingprofile_namespace_key";
-- modify "billing_profiles" table
-- atlas:nolint DS103 MF103
ALTER TABLE "billing_profiles" DROP COLUMN "provider_config", ADD COLUMN "metadata" jsonb NULL, ADD COLUMN "supplier_address_country" character varying NULL, ADD COLUMN "supplier_address_postal_code" character varying NULL, ADD COLUMN "supplier_address_state" character varying NULL, ADD COLUMN "supplier_address_city" character varying NULL, ADD COLUMN "supplier_address_line1" character varying NULL, ADD COLUMN "supplier_address_line2" character varying NULL, ADD COLUMN "supplier_address_phone_number" character varying NULL, ADD COLUMN "tax_provider" character varying NOT NULL, ADD COLUMN "invoicing_provider" character varying NOT NULL, ADD COLUMN "payment_provider" character varying NOT NULL, ADD COLUMN "supplier_name" character varying NOT NULL;
-- create index "billingprofile_namespace_id" to table: "billing_profiles"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billingprofile_namespace_id" ON "billing_profiles" ("namespace", "id");
-- create index "billing_profiles_workflow_config_id_key" to table: "billing_profiles"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billing_profiles_workflow_config_id_key" ON "billing_profiles" ("workflow_config_id");
-- create index "billingprofile_namespace_default_deleted_at" to table: "billing_profiles"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billingprofile_namespace_default_deleted_at" ON "billing_profiles" ("namespace", "default", "deleted_at");
-- create index "billingprofile_namespace_key_deleted_at" to table: "billing_profiles"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billingprofile_namespace_key_deleted_at" ON "billing_profiles" ("namespace", "key", "deleted_at");
-- modify "billing_workflow_configs" table
-- atlas:nolint BC102
ALTER TABLE "billing_workflow_configs" RENAME COLUMN "invoice_line_item_per_subject" TO "invoice_item_per_subject";
ALTER TABLE "billing_workflow_configs" ALTER COLUMN "invoice_item_per_subject" DROP DEFAULT;
-- rename a column from "alignment" to "collection_alignment"
-- atlas:nolint BC102
ALTER TABLE "billing_workflow_configs" RENAME COLUMN "alignment" TO "collection_alignment";
-- rename a column from "collection_period_seconds" to "item_collection_period_seconds"
-- atlas:nolint BC102
ALTER TABLE "billing_workflow_configs" RENAME COLUMN "collection_period_seconds" TO "item_collection_period_seconds";
-- rename a column from "invoice_line_item_resolution" to "invoice_item_resolution"
-- atlas:nolint BC102
ALTER TABLE "billing_workflow_configs" RENAME COLUMN "invoice_line_item_resolution" TO "invoice_item_resolution";
