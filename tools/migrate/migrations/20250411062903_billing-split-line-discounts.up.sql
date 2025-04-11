-- Sanity: make sure that there are no usage discounts in the database (it is still unsupported, but let's make sure)
DELETE FROM "billing_invoice_line_discounts" WHERE "type" = 'usage';
-- modify "billing_invoice_line_discounts" table
-- atlas:nolint MF104 CD101
ALTER TABLE "billing_invoice_line_discounts" DROP CONSTRAINT "billing_invoice_line_discounts_billing_invoice_lines_line_disco", ALTER COLUMN "amount" SET DEFAULT 0, ALTER COLUMN "amount" SET NOT NULL , ALTER COLUMN "type" DROP NOT NULL, ADD
 CONSTRAINT "billing_invoice_line_discounts_billing_invoice_lines_line_amoun" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- remove the default value for the amount column
ALTER TABLE "billing_invoice_line_discounts" ALTER COLUMN "amount" DROP DEFAULT;
-- create "billing_invoice_line_usage_discounts" table
CREATE TABLE "billing_invoice_line_usage_discounts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "child_unique_reference_id" character varying NULL,
  "description" character varying NULL,
  "reason" character varying NOT NULL,
  "invoicing_app_external_id" character varying NULL,
  "quantity" numeric NOT NULL,
  "pre_line_period_quantity" numeric NULL,
  "reason_details" jsonb NULL,
  "line_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_line_usage_discounts_billing_invoice_lines_line" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "billinginvoicelineusagediscount_id" to table: "billing_invoice_line_usage_discounts"
CREATE UNIQUE INDEX "billinginvoicelineusagediscount_id" ON "billing_invoice_line_usage_discounts" ("id");
-- create index "billinginvoicelineusagediscount_namespace" to table: "billing_invoice_line_usage_discounts"
CREATE INDEX "billinginvoicelineusagediscount_namespace" ON "billing_invoice_line_usage_discounts" ("namespace");
-- create index "billinginvoicelineusagediscount_namespace_line_id" to table: "billing_invoice_line_usage_discounts"
CREATE INDEX "billinginvoicelineusagediscount_namespace_line_id" ON "billing_invoice_line_usage_discounts" ("namespace", "line_id");
-- create index "billinginvoicelineusagediscount_namespace_line_id_child_unique_" to table: "billing_invoice_line_usage_discounts"
CREATE UNIQUE INDEX "billinginvoicelineusagediscount_namespace_line_id_child_unique_" ON "billing_invoice_line_usage_discounts" ("namespace", "line_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- drop "billing_invoice_discounts" table
-- atlas:nolint DS102
DROP TABLE "billing_invoice_discounts";
