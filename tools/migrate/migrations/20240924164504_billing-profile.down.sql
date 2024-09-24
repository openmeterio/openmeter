-- reverse: rename a column from "invoice_line_item_resolution" to "invoice_item_resolution"
ALTER TABLE "billing_workflow_configs" RENAME COLUMN "invoice_item_resolution" TO "invoice_line_item_resolution";
-- reverse: rename a column from "collection_period_seconds" to "item_collection_period_seconds"
ALTER TABLE "billing_workflow_configs" RENAME COLUMN "item_collection_period_seconds" TO "collection_period_seconds";
-- reverse: rename a column from "alignment" to "collection_alignment"
ALTER TABLE "billing_workflow_configs" RENAME COLUMN "collection_alignment" TO "alignment";
-- reverse: modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" DROP COLUMN "invoice_item_per_subject", DROP COLUMN "timezone", ADD COLUMN "invoice_line_item_per_subject" boolean NOT NULL DEFAULT false;
-- reverse: create index "billingprofile_namespace_default_deleted_at" to table: "billing_profiles"
DROP INDEX "billingprofile_namespace_default_deleted_at";
-- reverse: create index "billing_profiles_workflow_config_id_key" to table: "billing_profiles"
DROP INDEX "billing_profiles_workflow_config_id_key";
-- reverse: create index "billingprofile_namespace_id" to table: "billing_profiles"
DROP INDEX "billingprofile_namespace_id";
-- reverse: modify "billing_profiles" table
ALTER TABLE "billing_profiles" DROP COLUMN "supplier_name", DROP COLUMN "payment_provider", DROP COLUMN "invoicing_provider", DROP COLUMN "tax_provider", DROP COLUMN "supplier_address_phone_number", DROP COLUMN "supplier_address_line2", DROP COLUMN "supplier_address_line1", DROP COLUMN "supplier_address_city", DROP COLUMN "supplier_address_state", DROP COLUMN "supplier_address_postal_code", DROP COLUMN "supplier_address_country", DROP COLUMN "metadata", ADD COLUMN "provider_config" jsonb NOT NULL, ADD COLUMN "key" character varying NOT NULL;
-- reverse: drop index "billingprofile_namespace_id" from table: "billing_profiles"
CREATE INDEX "billingprofile_namespace_id" ON "billing_profiles" ("namespace", "id");
-- reverse: drop index "billingprofile_namespace_default" from table: "billing_profiles"
CREATE INDEX "billingprofile_namespace_default" ON "billing_profiles" ("namespace", "default");
-- reverse: create index "billing_invoices_workflow_config_id_key" to table: "billing_invoices"
DROP INDEX "billing_invoices_workflow_config_id_key";
-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" DROP COLUMN "payment_provider", DROP COLUMN "invoicing_provider", DROP COLUMN "tax_provider", ADD COLUMN "provider_reference" jsonb NOT NULL, ADD COLUMN "provider_config" jsonb NOT NULL;
