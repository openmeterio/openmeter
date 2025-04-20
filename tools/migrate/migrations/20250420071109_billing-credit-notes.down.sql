-- reverse: create index "billinginvoicecreditnoteline_namespace_parent_line_id_child_uni" to table: "billing_invoice_credit_note_lines"
DROP INDEX "billinginvoicecreditnoteline_namespace_parent_line_id_child_uni";
-- reverse: create index "billinginvoicecreditnoteline_namespace_invoice_id" to table: "billing_invoice_credit_note_lines"
DROP INDEX "billinginvoicecreditnoteline_namespace_invoice_id";
-- reverse: create index "billinginvoicecreditnoteline_namespace_id" to table: "billing_invoice_credit_note_lines"
DROP INDEX "billinginvoicecreditnoteline_namespace_id";
-- reverse: create index "billinginvoicecreditnoteline_namespace" to table: "billing_invoice_credit_note_lines"
DROP INDEX "billinginvoicecreditnoteline_namespace";
-- reverse: create index "billinginvoicecreditnoteline_id" to table: "billing_invoice_credit_note_lines"
DROP INDEX "billinginvoicecreditnoteline_id";
-- reverse: create "billing_invoice_credit_note_lines" table
DROP TABLE "billing_invoice_credit_note_lines";
