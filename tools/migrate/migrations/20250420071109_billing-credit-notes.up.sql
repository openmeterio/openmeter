-- create "billing_invoice_credit_note_lines" table
CREATE TABLE "billing_invoice_credit_note_lines" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "amount" numeric NOT NULL,
  "taxes_total" numeric NOT NULL,
  "taxes_inclusive_total" numeric NOT NULL,
  "taxes_exclusive_total" numeric NOT NULL,
  "charges_total" numeric NOT NULL,
  "discounts_total" numeric NOT NULL,
  "total" numeric NOT NULL,
  "managed_by" character varying NOT NULL,
  "period_start" timestamptz NOT NULL,
  "period_end" timestamptz NOT NULL,
  "invoice_at" timestamptz NOT NULL,
  "status" character varying NOT NULL,
  "currency" character varying(3) NOT NULL,
  "invoicing_app_external_id" character varying NULL,
  "child_unique_reference_id" character varying NULL,
  "credit_note_amount" numeric NOT NULL,
  "tax_config" jsonb NULL,
  "invoice_id" character(26) NOT NULL,
  "parent_line_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_credit_note_lines_billing_invoice_lines_billing" FOREIGN KEY ("parent_line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "billing_invoice_credit_note_lines_billing_invoices_billing_invo" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "billinginvoicecreditnoteline_id" to table: "billing_invoice_credit_note_lines"
CREATE UNIQUE INDEX "billinginvoicecreditnoteline_id" ON "billing_invoice_credit_note_lines" ("id");
-- create index "billinginvoicecreditnoteline_namespace" to table: "billing_invoice_credit_note_lines"
CREATE INDEX "billinginvoicecreditnoteline_namespace" ON "billing_invoice_credit_note_lines" ("namespace");
-- create index "billinginvoicecreditnoteline_namespace_id" to table: "billing_invoice_credit_note_lines"
CREATE UNIQUE INDEX "billinginvoicecreditnoteline_namespace_id" ON "billing_invoice_credit_note_lines" ("namespace", "id");
-- create index "billinginvoicecreditnoteline_namespace_invoice_id" to table: "billing_invoice_credit_note_lines"
CREATE INDEX "billinginvoicecreditnoteline_namespace_invoice_id" ON "billing_invoice_credit_note_lines" ("namespace", "invoice_id");
-- create index "billinginvoicecreditnoteline_namespace_parent_line_id_child_uni" to table: "billing_invoice_credit_note_lines"
CREATE UNIQUE INDEX "billinginvoicecreditnoteline_namespace_parent_line_id_child_uni" ON "billing_invoice_credit_note_lines" ("namespace", "parent_line_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
