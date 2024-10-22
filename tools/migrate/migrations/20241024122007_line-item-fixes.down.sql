-- reverse: drop "billing_invoice_items" table
CREATE TABLE "billing_invoice_items" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "metadata" jsonb NULL,
  "customer_id" character(26) NOT NULL,
  "period_start" timestamptz NOT NULL,
  "period_end" timestamptz NOT NULL,
  "invoice_at" timestamptz NOT NULL,
  "quantity" numeric NULL,
  "unit_price" numeric NOT NULL,
  "currency" character varying(3) NOT NULL,
  "tax_code_override" jsonb NOT NULL,
  "invoice_id" character(26) NULL,
  "type" character varying NOT NULL,
  "name" character varying NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_items_billing_invoices_billing_invoice_items" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
CREATE UNIQUE INDEX "billinginvoiceitem_id" ON "billing_invoice_items" ("id");
CREATE INDEX "billinginvoiceitem_namespace" ON "billing_invoice_items" ("namespace");
CREATE INDEX "billinginvoiceitem_namespace_customer_id" ON "billing_invoice_items" ("namespace", "customer_id");
CREATE INDEX "billinginvoiceitem_namespace_id" ON "billing_invoice_items" ("namespace", "id");
CREATE INDEX "billinginvoiceitem_namespace_invoice_id" ON "billing_invoice_items" ("namespace", "invoice_id");
-- reverse: modify "billing_profiles" table
ALTER TABLE "billing_profiles" DROP CONSTRAINT "billing_profiles_apps_billing_profile_tax_app", DROP CONSTRAINT "billing_profiles_apps_billing_profile_payment_app", DROP CONSTRAINT "billing_profiles_apps_billing_profile_invoicing_app", ADD
 CONSTRAINT "billing_profiles_apps_tax_app" FOREIGN KEY ("tax_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD
 CONSTRAINT "billing_profiles_apps_payment_app" FOREIGN KEY ("payment_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD
 CONSTRAINT "billing_profiles_apps_invoicing_app" FOREIGN KEY ("invoicing_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- reverse: create index "billinginvoiceline_namespace_invoice_id" to table: "billing_invoice_lines"
DROP INDEX "billinginvoiceline_namespace_invoice_id";
-- reverse: create index "billinginvoiceline_namespace_id" to table: "billing_invoice_lines"
DROP INDEX "billinginvoiceline_namespace_id";
-- reverse: create index "billinginvoiceline_namespace" to table: "billing_invoice_lines"
DROP INDEX "billinginvoiceline_namespace";
-- reverse: create index "billinginvoiceline_id" to table: "billing_invoice_lines"
DROP INDEX "billinginvoiceline_id";
-- reverse: create "billing_invoice_lines" table
DROP TABLE "billing_invoice_lines";
-- reverse: rename a column from "code" to "number"
ALTER TABLE "billing_invoices" RENAME COLUMN "number" TO "code";
-- reverse: rename a column from "billing_profile_id" to "source_billing_profile_id"
ALTER TABLE "billing_invoices" RENAME COLUMN "source_billing_profile_id" TO "billing_profile_id";
-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" DROP CONSTRAINT "billing_invoices_customers_billing_invoice", DROP CONSTRAINT "billing_invoices_apps_billing_invoice_tax_app", DROP CONSTRAINT "billing_invoices_apps_billing_invoice_payment_app", DROP CONSTRAINT "billing_invoices_apps_billing_invoice_invoicing_app", DROP COLUMN "payment_app_id", DROP COLUMN "invoicing_app_id", DROP COLUMN "tax_app_id", DROP COLUMN "due_at", DROP COLUMN "issued_at", DROP COLUMN "description", DROP COLUMN "type", DROP COLUMN "customer_timezone", DROP COLUMN "customer_name", DROP COLUMN "supplier_tax_code", DROP COLUMN "supplier_name", DROP COLUMN "customer_address_phone_number", DROP COLUMN "customer_address_line2", DROP COLUMN "customer_address_line1", DROP COLUMN "customer_address_city", DROP COLUMN "customer_address_state", DROP COLUMN "customer_address_postal_code", DROP COLUMN "customer_address_country", DROP COLUMN "supplier_address_phone_number", DROP COLUMN "supplier_address_line2", DROP COLUMN "supplier_address_line1", DROP COLUMN "supplier_address_city", DROP COLUMN "supplier_address_state", DROP COLUMN "supplier_address_postal_code", DROP COLUMN "supplier_address_country", ADD COLUMN "series" character varying NULL, ALTER COLUMN "period_end" SET NOT NULL, ALTER COLUMN "period_start" SET NOT NULL, ADD COLUMN "due_date" timestamptz NOT NULL;
-- reverse: drop index "billinginvoice_namespace_status" from table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_status" ON "billing_invoices" ("namespace", "status");
-- reverse: create index "billinginvoicemanuallineconfig_namespace" to table: "billing_invoice_manual_line_configs"
DROP INDEX "billinginvoicemanuallineconfig_namespace";
-- reverse: create index "billinginvoicemanuallineconfig_id" to table: "billing_invoice_manual_line_configs"
DROP INDEX "billinginvoicemanuallineconfig_id";
-- reverse: create "billing_invoice_manual_line_configs" table
DROP TABLE "billing_invoice_manual_line_configs";
-- reverse: rename a column from "item_collection_period" to "line_collection_period"
ALTER TABLE "billing_workflow_configs" RENAME COLUMN "line_collection_period" TO "item_collection_period";
-- reverse: rename a column from "item_collection_period" to "line_collection_period"
ALTER TABLE "billing_customer_overrides" RENAME COLUMN "line_collection_period" TO "item_collection_period";
