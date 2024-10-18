-- modify "billing_customer_overrides" table
-- atlas:nolint DS103
ALTER TABLE "billing_customer_overrides" DROP COLUMN "item_collection_period_seconds", DROP COLUMN "invoice_draft_period_seconds", DROP COLUMN "invoice_due_after_seconds", DROP COLUMN "invoice_item_resolution", DROP COLUMN "invoice_item_per_subject", ADD COLUMN "item_collection_period" character varying NULL, ADD COLUMN "invoice_draft_period" character varying NULL, ADD COLUMN "invoice_due_after" character varying NULL;
-- modify "billing_invoices" table
-- atlas:nolint DS103
ALTER TABLE "billing_invoices" DROP COLUMN "tax_provider", DROP COLUMN "invoicing_provider", DROP COLUMN "payment_provider";
-- modify "billing_workflow_configs" table
-- atlas:nolint DS103 MF103
ALTER TABLE "billing_workflow_configs" DROP COLUMN "item_collection_period_seconds", DROP COLUMN "invoice_draft_period_seconds", DROP COLUMN "invoice_due_after_seconds", DROP COLUMN "invoice_item_resolution", DROP COLUMN "timezone", DROP COLUMN "invoice_item_per_subject", ADD COLUMN "item_collection_period" character varying NOT NULL, ADD COLUMN "invoice_draft_period" character varying NOT NULL, ADD COLUMN "invoice_due_after" character varying NOT NULL;
-- modify "billing_profiles" table
-- atlas:nolint DS103 MF103
ALTER TABLE "billing_profiles" DROP COLUMN "tax_provider", DROP COLUMN "invoicing_provider", DROP COLUMN "payment_provider", ADD COLUMN "supplier_tax_code" character varying NULL, ADD COLUMN "tax_app_id" character(26) NOT NULL, ADD COLUMN "invoicing_app_id" character(26) NOT NULL, ADD COLUMN "payment_app_id" character(26) NOT NULL, ADD
 CONSTRAINT "billing_profiles_apps_invoicing_app" FOREIGN KEY ("invoicing_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD
 CONSTRAINT "billing_profiles_apps_payment_app" FOREIGN KEY ("payment_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD
 CONSTRAINT "billing_profiles_apps_tax_app" FOREIGN KEY ("tax_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
