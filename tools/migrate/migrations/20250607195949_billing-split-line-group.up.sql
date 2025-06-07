-- create "billing_invoice_split_line_groups" table
CREATE TABLE "billing_invoice_split_line_groups" (
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
  "currency" character varying(3) NOT NULL,
  "tax_config" jsonb NULL,
  "child_unique_reference_id" character varying NULL,
  "ratecard_discounts" jsonb NULL,
  "price" jsonb NOT NULL,
  "customer_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "billing_invoice_split_line_groups_customers_billing_split_line_" FOREIGN KEY ("customer_id") REFERENCES "customers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create index "billinginvoicesplitlinegroup_id" to table: "billing_invoice_split_line_groups"
CREATE UNIQUE INDEX "billinginvoicesplitlinegroup_id" ON "billing_invoice_split_line_groups" ("id");
-- create index "billinginvoicesplitlinegroup_namespace" to table: "billing_invoice_split_line_groups"
CREATE INDEX "billinginvoicesplitlinegroup_namespace" ON "billing_invoice_split_line_groups" ("namespace");
-- create index "billinginvoicesplitlinegroup_namespace_customer_id_child_unique" to table: "billing_invoice_split_line_groups"
CREATE UNIQUE INDEX "billinginvoicesplitlinegroup_namespace_customer_id_child_unique" ON "billing_invoice_split_line_groups" ("namespace", "customer_id", "child_unique_reference_id") WHERE ((child_unique_reference_id IS NOT NULL) AND (deleted_at IS NULL));
-- create index "billinginvoicesplitlinegroup_namespace_id" to table: "billing_invoice_split_line_groups"
CREATE UNIQUE INDEX "billinginvoicesplitlinegroup_namespace_id" ON "billing_invoice_split_line_groups" ("namespace", "id");
-- modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" ADD COLUMN "split_line_group_id" character(26) NULL, ADD CONSTRAINT "billing_invoice_lines_billing_invoice_split_line_groups_billing" FOREIGN KEY ("split_line_group_id") REFERENCES "billing_invoice_split_line_groups" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
