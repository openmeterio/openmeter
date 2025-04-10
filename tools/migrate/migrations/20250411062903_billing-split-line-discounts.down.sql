-- reverse: drop "billing_invoice_discounts" table
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
  PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "billinginvoicediscount_id" ON "billing_invoice_discounts" ("id");
CREATE INDEX "billinginvoicediscount_namespace" ON "billing_invoice_discounts" ("namespace");
CREATE UNIQUE INDEX "billinginvoicediscount_namespace_id" ON "billing_invoice_discounts" ("namespace", "id");
CREATE INDEX "billinginvoicediscount_namespace_invoice_id" ON "billing_invoice_discounts" ("namespace", "invoice_id");
-- reverse: create index "billinginvoicelineusagediscount_namespace_line_id_child_unique_" to table: "billing_invoice_line_usage_discounts"
DROP INDEX "billinginvoicelineusagediscount_namespace_line_id_child_unique_";
-- reverse: create index "billinginvoicelineusagediscount_namespace_line_id" to table: "billing_invoice_line_usage_discounts"
DROP INDEX "billinginvoicelineusagediscount_namespace_line_id";
-- reverse: create index "billinginvoicelineusagediscount_namespace" to table: "billing_invoice_line_usage_discounts"
DROP INDEX "billinginvoicelineusagediscount_namespace";
-- reverse: create index "billinginvoicelineusagediscount_id" to table: "billing_invoice_line_usage_discounts"
DROP INDEX "billinginvoicelineusagediscount_id";
-- reverse: create "billing_invoice_line_usage_discounts" table
DROP TABLE "billing_invoice_line_usage_discounts";
-- reverse: modify "billing_invoice_line_discounts" table
ALTER TABLE "billing_invoice_line_discounts" DROP CONSTRAINT "billing_invoice_line_discounts_billing_invoice_lines_line_amoun", ALTER COLUMN "type" SET NOT NULL, ALTER COLUMN "amount" DROP NOT NULL, ADD
 CONSTRAINT "billing_invoice_line_discounts_billing_invoice_lines_line_disco" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
