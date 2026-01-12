-- create "billing_invoice_write_schema_levels" table
CREATE TABLE "billing_invoice_write_schema_levels" (
  "id" character varying NOT NULL,
  "schema_level" bigint NOT NULL,
  PRIMARY KEY ("id")
);

INSERT INTO "billing_invoice_write_schema_levels" ("id", "schema_level") VALUES ('write_schema_level', 1);
-- create index "billinginvoicewriteschemalevel_id" to table: "billing_invoice_write_schema_levels"
CREATE UNIQUE INDEX "billinginvoicewriteschemalevel_id" ON "billing_invoice_write_schema_levels" ("id");
-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "schema_level" bigint NOT NULL DEFAULT 1;
-- create "billing_standard_invoice_detailed_lines" table
CREATE TABLE "billing_standard_invoice_detailed_lines" (
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
  "service_period_start" timestamptz NOT NULL,
  "service_period_end" timestamptz NOT NULL,
  "quantity" numeric NOT NULL,
  "invoicing_app_external_id" character varying NULL,
  "child_unique_reference_id" character varying NULL,
  "per_unit_amount" numeric NOT NULL,
  "category" character varying NOT NULL DEFAULT 'regular',
  "payment_term" character varying NOT NULL DEFAULT 'in_advance',
  "index" bigint NULL,
  "invoice_id" character(26) NOT NULL,
  "parent_line_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_standard_invoice_detailed_lines_billing_invoice_lines_d" FOREIGN KEY ("parent_line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "billing_standard_invoice_detailed_lines_billing_invoices_billin" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "billingstandardinvoicedetailedline_annotations" to table: "billing_standard_invoice_detailed_lines"
CREATE INDEX "billingstandardinvoicedetailedline_annotations" ON "billing_standard_invoice_detailed_lines" USING gin ("annotations");
-- create index "billingstandardinvoicedetailedline_id" to table: "billing_standard_invoice_detailed_lines"
CREATE UNIQUE INDEX "billingstandardinvoicedetailedline_id" ON "billing_standard_invoice_detailed_lines" ("id");
-- create index "billingstandardinvoicedetailedline_namespace" to table: "billing_standard_invoice_detailed_lines"
CREATE INDEX "billingstandardinvoicedetailedline_namespace" ON "billing_standard_invoice_detailed_lines" ("namespace");
-- create index "billingstandardinvoicedetailedline_namespace_id" to table: "billing_standard_invoice_detailed_lines"
CREATE UNIQUE INDEX "billingstandardinvoicedetailedline_namespace_id" ON "billing_standard_invoice_detailed_lines" ("namespace", "id");
-- create index "billingstandardinvoicedetailedline_namespace_invoice_id" to table: "billing_standard_invoice_detailed_lines"
CREATE INDEX "billingstandardinvoicedetailedline_namespace_invoice_id" ON "billing_standard_invoice_detailed_lines" ("namespace", "invoice_id");
-- create index "billingstandardinvoicedetailedline_namespace_parent_line_id" to table: "billing_standard_invoice_detailed_lines"
CREATE INDEX "billingstandardinvoicedetailedline_namespace_parent_line_id" ON "billing_standard_invoice_detailed_lines" ("namespace", "parent_line_id");
-- create index "billingstdinvdetailedline_ns_parent_child_id" to table: "billing_standard_invoice_detailed_lines"
CREATE UNIQUE INDEX "billingstdinvdetailedline_ns_parent_child_id" ON "billing_standard_invoice_detailed_lines" ("namespace", "parent_line_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- create "billing_standard_invoice_detailed_line_amount_discounts" table
CREATE TABLE "billing_standard_invoice_detailed_line_amount_discounts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "child_unique_reference_id" character varying NULL,
  "description" character varying NULL,
  "reason" character varying NOT NULL,
  "invoicing_app_external_id" character varying NULL,
  "amount" numeric NOT NULL,
  "rounding_amount" numeric NULL,
  "source_discount" jsonb NULL,
  "line_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_standard_invoice_detailed_line_amount_discounts_billing" FOREIGN KEY ("line_id") REFERENCES "billing_standard_invoice_detailed_lines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "billingstandardinvoicedetailedlineamountdiscount_id" to table: "billing_standard_invoice_detailed_line_amount_discounts"
CREATE UNIQUE INDEX "billingstandardinvoicedetailedlineamountdiscount_id" ON "billing_standard_invoice_detailed_line_amount_discounts" ("id");
-- create index "billingstandardinvoicedetailedlineamountdiscount_namespace" to table: "billing_standard_invoice_detailed_line_amount_discounts"
CREATE INDEX "billingstandardinvoicedetailedlineamountdiscount_namespace" ON "billing_standard_invoice_detailed_line_amount_discounts" ("namespace");
-- create index "billingstandardinvoicedetailedlineamountdiscount_namespace_line" to table: "billing_standard_invoice_detailed_line_amount_discounts"
CREATE INDEX "billingstandardinvoicedetailedlineamountdiscount_namespace_line" ON "billing_standard_invoice_detailed_line_amount_discounts" ("namespace", "line_id");
-- create index "billingstdinvdetailedlineamntdiscount_ns_parent_child_id" to table: "billing_standard_invoice_detailed_line_amount_discounts"
CREATE UNIQUE INDEX "billingstdinvdetailedlineamntdiscount_ns_parent_child_id" ON "billing_standard_invoice_detailed_line_amount_discounts" ("namespace", "line_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
