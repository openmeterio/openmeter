-- rename a column from "item_collection_period" to "line_collection_period"
-- atlas:nolint BC102
ALTER TABLE "billing_customer_overrides" RENAME COLUMN "item_collection_period" TO "line_collection_period";
-- rename a column from "item_collection_period" to "line_collection_period"
-- atlas:nolint BC102
ALTER TABLE "billing_workflow_configs" RENAME COLUMN "item_collection_period" TO "line_collection_period";
-- create "billing_invoice_manual_line_configs" table
CREATE TABLE "billing_invoice_manual_line_configs" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "unit_price" numeric NOT NULL,
  PRIMARY KEY ("id")
);
-- create index "billinginvoicemanuallineconfig_id" to table: "billing_invoice_manual_line_configs"
CREATE UNIQUE INDEX "billinginvoicemanuallineconfig_id" ON "billing_invoice_manual_line_configs" ("id");
-- create index "billinginvoicemanuallineconfig_namespace" to table: "billing_invoice_manual_line_configs"
CREATE INDEX "billinginvoicemanuallineconfig_namespace" ON "billing_invoice_manual_line_configs" ("namespace");
-- drop index "billinginvoice_namespace_status" from table: "billing_invoices"
DROP INDEX "billinginvoice_namespace_status";
-- modify "billing_invoices" table
-- atlas:nolint DS103 MF103
ALTER TABLE "billing_invoices" DROP COLUMN "due_date", ALTER COLUMN "period_start" DROP NOT NULL, ALTER COLUMN "period_end" DROP NOT NULL, DROP COLUMN "series", ADD COLUMN "supplier_address_country" character varying NULL, ADD COLUMN "supplier_address_postal_code" character varying NULL, ADD COLUMN "supplier_address_state" character varying NULL, ADD COLUMN "supplier_address_city" character varying NULL, ADD COLUMN "supplier_address_line1" character varying NULL, ADD COLUMN "supplier_address_line2" character varying NULL, ADD COLUMN "supplier_address_phone_number" character varying NULL, ADD COLUMN "customer_address_country" character varying NULL, ADD COLUMN "customer_address_postal_code" character varying NULL, ADD COLUMN "customer_address_state" character varying NULL, ADD COLUMN "customer_address_city" character varying NULL, ADD COLUMN "customer_address_line1" character varying NULL, ADD COLUMN "customer_address_line2" character varying NULL, ADD COLUMN "customer_address_phone_number" character varying NULL, ADD COLUMN "supplier_name" character varying NOT NULL, ADD COLUMN "supplier_tax_code" character varying NULL, ADD COLUMN "customer_name" character varying NOT NULL, ADD COLUMN "customer_timezone" character varying NULL, ADD COLUMN "type" character varying NOT NULL, ADD COLUMN "description" character varying NULL, ADD COLUMN "issued_at" timestamptz NULL, ADD COLUMN "due_at" timestamptz NULL, ADD COLUMN "tax_app_id" character(26) NOT NULL, ADD COLUMN "invoicing_app_id" character(26) NOT NULL, ADD COLUMN "payment_app_id" character(26) NOT NULL, ADD
 CONSTRAINT "billing_invoices_apps_billing_invoice_invoicing_app" FOREIGN KEY ("invoicing_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD
 CONSTRAINT "billing_invoices_apps_billing_invoice_payment_app" FOREIGN KEY ("payment_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD
 CONSTRAINT "billing_invoices_apps_billing_invoice_tax_app" FOREIGN KEY ("tax_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD
 CONSTRAINT "billing_invoices_customers_billing_invoice" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- rename a column from "billing_profile_id" to "source_billing_profile_id"
-- atlas:nolint BC102
ALTER TABLE "billing_invoices" RENAME COLUMN "billing_profile_id" TO "source_billing_profile_id";
-- rename a column from "code" to "number"
-- atlas:nolint BC102
ALTER TABLE "billing_invoices" RENAME COLUMN "code" TO "number";
-- create "billing_invoice_lines" table
CREATE TABLE "billing_invoice_lines" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "period_start" timestamptz NOT NULL,
  "period_end" timestamptz NOT NULL,
  "invoice_at" timestamptz NOT NULL,
  "type" character varying NOT NULL,
  "status" character varying NOT NULL,
  "currency" character varying(3) NOT NULL,
  "quantity" numeric NULL,
  "tax_overrides" jsonb NULL,
  "invoice_id" character(26) NOT NULL,
  "manual_line_config_id" character(26) NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_lines_billing_invoice_manual_line_configs_billi" FOREIGN KEY ("manual_line_config_id") REFERENCES "billing_invoice_manual_line_configs" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "billing_invoice_lines_billing_invoices_billing_invoice_lines" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "billinginvoiceline_id" to table: "billing_invoice_lines"
CREATE UNIQUE INDEX "billinginvoiceline_id" ON "billing_invoice_lines" ("id");
-- create index "billinginvoiceline_namespace" to table: "billing_invoice_lines"
CREATE INDEX "billinginvoiceline_namespace" ON "billing_invoice_lines" ("namespace");
-- create index "billinginvoiceline_namespace_id" to table: "billing_invoice_lines"
CREATE UNIQUE INDEX "billinginvoiceline_namespace_id" ON "billing_invoice_lines" ("namespace", "id");
-- create index "billinginvoiceline_namespace_invoice_id" to table: "billing_invoice_lines"
CREATE INDEX "billinginvoiceline_namespace_invoice_id" ON "billing_invoice_lines" ("namespace", "invoice_id");
-- modify "billing_profiles" table
-- atlas:nolint CD101
ALTER TABLE "billing_profiles" DROP CONSTRAINT "billing_profiles_apps_invoicing_app", DROP CONSTRAINT "billing_profiles_apps_payment_app", DROP CONSTRAINT "billing_profiles_apps_tax_app", ADD
 CONSTRAINT "billing_profiles_apps_billing_profile_invoicing_app" FOREIGN KEY ("invoicing_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD
 CONSTRAINT "billing_profiles_apps_billing_profile_payment_app" FOREIGN KEY ("payment_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION, ADD
 CONSTRAINT "billing_profiles_apps_billing_profile_tax_app" FOREIGN KEY ("tax_app_id") REFERENCES "apps" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- drop "billing_invoice_items" table
-- atlas:nolint DS102
DROP TABLE "billing_invoice_items";
