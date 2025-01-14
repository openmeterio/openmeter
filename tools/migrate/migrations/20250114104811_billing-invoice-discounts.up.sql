-- create "billing_invoice_discounts" table
CREATE TABLE "billing_invoice_discounts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "type" character varying NOT NULL,
  "amount" numeric NOT NULL,
  "line_ids" jsonb NULL,
  "invoice_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_discounts_billing_invoices_invoice_discounts" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "billinginvoicediscount_id" to table: "billing_invoice_discounts"
CREATE UNIQUE INDEX "billinginvoicediscount_id" ON "billing_invoice_discounts" ("id");
-- create index "billinginvoicediscount_namespace" to table: "billing_invoice_discounts"
CREATE INDEX "billinginvoicediscount_namespace" ON "billing_invoice_discounts" ("namespace");
-- create index "billinginvoicediscount_namespace_id" to table: "billing_invoice_discounts"
CREATE UNIQUE INDEX "billinginvoicediscount_namespace_id" ON "billing_invoice_discounts" ("namespace", "id");
-- create index "billinginvoicediscount_namespace_invoice_id" to table: "billing_invoice_discounts"
CREATE INDEX "billinginvoicediscount_namespace_invoice_id" ON "billing_invoice_discounts" ("namespace", "invoice_id");
-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "line_ids" character(26) NULL, ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_discounts_lines" FOREIGN KEY ("line_ids") REFERENCES "billing_invoice_discounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
