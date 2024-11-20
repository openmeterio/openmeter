-- rename a column from "amount" to "per_unit_amount"
-- atlas:nolint BC102
ALTER TABLE "billing_invoice_flat_fee_line_configs" RENAME COLUMN "amount" TO "per_unit_amount";
-- modify "billing_invoice_lines" table
-- atlas:nolint CD101
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_billing_invoice_lines_child_lines", ADD COLUMN "child_unique_reference_id" character varying NULL, ADD
 CONSTRAINT "billing_invoice_lines_billing_invoice_lines_detailed_lines" FOREIGN KEY ("parent_line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "billinginvoiceline_namespace_parent_line_id_child_unique_refere" to table: "billing_invoice_lines"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billinginvoiceline_namespace_parent_line_id_child_unique_refere" ON "billing_invoice_lines" ("namespace", "parent_line_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- create "billing_invoice_line_discounts" table
CREATE TABLE "billing_invoice_line_discounts" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "child_unique_reference_id" character varying NULL,
  "description" character varying NULL,
  "amount" numeric NOT NULL,
  "line_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_line_discounts_billing_invoice_lines_line_disco" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "billinginvoicelinediscount_id" to table: "billing_invoice_line_discounts"
CREATE UNIQUE INDEX "billinginvoicelinediscount_id" ON "billing_invoice_line_discounts" ("id");
-- create index "billinginvoicelinediscount_namespace" to table: "billing_invoice_line_discounts"
CREATE INDEX "billinginvoicelinediscount_namespace" ON "billing_invoice_line_discounts" ("namespace");
-- create index "billinginvoicelinediscount_namespace_line_id" to table: "billing_invoice_line_discounts"
CREATE INDEX "billinginvoicelinediscount_namespace_line_id" ON "billing_invoice_line_discounts" ("namespace", "line_id");
-- create index "billinginvoicelinediscount_namespace_line_id_child_unique_refer" to table: "billing_invoice_line_discounts"
CREATE UNIQUE INDEX "billinginvoicelinediscount_namespace_line_id_child_unique_refer" ON "billing_invoice_line_discounts" ("namespace", "line_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
