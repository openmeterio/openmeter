-- create "billing_invoice_validation_issues" table
CREATE TABLE "billing_invoice_validation_issues" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "severity" character varying NOT NULL,
  "code" character varying NULL,
  "message" character varying NOT NULL,
  "path" character varying NULL,
  "component" character varying NOT NULL,
  "dedupe_hash" bytea NOT NULL,
  "invoice_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_validation_issues_billing_invoices_billing_invo" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "billinginvoicevalidationissue_id" to table: "billing_invoice_validation_issues"
CREATE UNIQUE INDEX "billinginvoicevalidationissue_id" ON "billing_invoice_validation_issues" ("id");
-- create index "billinginvoicevalidationissue_namespace" to table: "billing_invoice_validation_issues"
CREATE INDEX "billinginvoicevalidationissue_namespace" ON "billing_invoice_validation_issues" ("namespace");
-- create index "billinginvoicevalidationissue_namespace_invoice_id_dedupe_hash" to table: "billing_invoice_validation_issues"
CREATE UNIQUE INDEX "billinginvoicevalidationissue_namespace_invoice_id_dedupe_hash" ON "billing_invoice_validation_issues" ("namespace", "invoice_id", "dedupe_hash");
