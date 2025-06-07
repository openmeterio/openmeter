-- reverse: modify "billing_invoice_lines" table
ALTER TABLE "billing_invoice_lines" DROP CONSTRAINT "billing_invoice_lines_billing_invoice_split_line_groups_billing", DROP COLUMN "split_line_group_id";
-- reverse: create index "billinginvoicesplitlinegroup_namespace_id" to table: "billing_invoice_split_line_groups"
DROP INDEX "billinginvoicesplitlinegroup_namespace_id";
-- reverse: create index "billinginvoicesplitlinegroup_namespace_customer_id_child_unique" to table: "billing_invoice_split_line_groups"
DROP INDEX "billinginvoicesplitlinegroup_namespace_customer_id_child_unique";
-- reverse: create index "billinginvoicesplitlinegroup_namespace" to table: "billing_invoice_split_line_groups"
DROP INDEX "billinginvoicesplitlinegroup_namespace";
-- reverse: create index "billinginvoicesplitlinegroup_id" to table: "billing_invoice_split_line_groups"
DROP INDEX "billinginvoicesplitlinegroup_id";
-- reverse: create "billing_invoice_split_line_groups" table
DROP TABLE "billing_invoice_split_line_groups";
