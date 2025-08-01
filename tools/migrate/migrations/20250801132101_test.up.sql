-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_billing_invoice_lines_detailed_lines", DROP COLUMN "fee_line_config_id";
-- create "billing_invoice_detailed_lines" table
CREATE TABLE "billing_invoice_detailed_lines" (
  "id" character(26) NOT NULL,
  "annotations" jsonb NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "currency" character varying(3) NOT NULL,
  "tax_config" jsonb NULL,
  "amount" numeric NOT NULL,
  "taxes_total" numeric NOT NULL,
  "taxes_inclusive_total" numeric NOT NULL,
  "taxes_exclusive_total" numeric NOT NULL,
  "charges_total" numeric NOT NULL,
  "discounts_total" numeric NOT NULL,
  "total" numeric NOT NULL,
  "service_period_from" timestamptz NOT NULL,
  "service_period_to" timestamptz NOT NULL,
  "quantity" numeric NOT NULL,
  "per_unit_amount" numeric NOT NULL,
  "category" character varying NOT NULL,
  "payment_term" character varying NOT NULL,
  "invoicing_app_external_id" character varying NULL,
  "child_unique_reference_id" character varying NULL,
  "index" bigint NULL,
  "invoice_id" character(26) NOT NULL,
  "parent_line_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_detailed_lines_billing_invoice_lines_detailed_l" FOREIGN KEY ("parent_line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "billing_invoice_detailed_lines_billing_invoices_detailed_lines" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "billinginvoicedetailedline_annotations" to table: "billing_invoice_detailed_lines"
CREATE INDEX "billinginvoicedetailedline_annotations" ON "billing_invoice_detailed_lines" USING gin ("annotations");
-- create index "billinginvoicedetailedline_id" to table: "billing_invoice_detailed_lines"
CREATE UNIQUE INDEX "billinginvoicedetailedline_id" ON "billing_invoice_detailed_lines" ("id");
-- create index "billinginvoicedetailedline_namespace" to table: "billing_invoice_detailed_lines"
CREATE INDEX "billinginvoicedetailedline_namespace" ON "billing_invoice_detailed_lines" ("namespace");
-- create index "billinginvoicedetailedline_namespace_id" to table: "billing_invoice_detailed_lines"
CREATE UNIQUE INDEX "billinginvoicedetailedline_namespace_id" ON "billing_invoice_detailed_lines" ("namespace", "id");
-- create index "billinginvoicedetailedline_namespace_invoice_id" to table: "billing_invoice_detailed_lines"
CREATE INDEX "billinginvoicedetailedline_namespace_invoice_id" ON "billing_invoice_detailed_lines" ("namespace", "invoice_id");
-- create index "billinginvoicedetailedline_namespace_parent_line_id" to table: "billing_invoice_detailed_lines"
CREATE INDEX "billinginvoicedetailedline_namespace_parent_line_id" ON "billing_invoice_detailed_lines" ("namespace", "parent_line_id");
-- create index "billinginvoicedetailedline_namespace_parent_line_id_child_uniqu" to table: "billing_invoice_detailed_lines"
CREATE UNIQUE INDEX "billinginvoicedetailedline_namespace_parent_line_id_child_uniqu" ON "billing_invoice_detailed_lines" ("namespace", "parent_line_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
