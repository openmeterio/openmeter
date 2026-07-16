-- reverse: create index "billinggatheringline_ns_subscription_ref" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringline_ns_subscription_ref";
-- reverse: create index "billinggatheringline_ns_invoice_child_id" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringline_ns_invoice_child_id";
-- reverse: create index "billinggatheringinvoiceline_tax_code_id" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringinvoiceline_tax_code_id";
-- reverse: create index "billinggatheringinvoiceline_namespace_split_line_group_id" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringinvoiceline_namespace_split_line_group_id";
-- reverse: create index "billinggatheringinvoiceline_namespace_invoice_id" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringinvoiceline_namespace_invoice_id";
-- reverse: create index "billinggatheringinvoiceline_namespace_id" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringinvoiceline_namespace_id";
-- reverse: create index "billinggatheringinvoiceline_namespace_charge_id" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringinvoiceline_namespace_charge_id";
-- reverse: create index "billinggatheringinvoiceline_namespace" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringinvoiceline_namespace";
-- reverse: create index "billinggatheringinvoiceline_id" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringinvoiceline_id";
-- reverse: create index "billinggatheringinvoiceline_annotations" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringinvoiceline_annotations";
-- reverse: create "billing_gathering_invoice_lines" table
DROP TABLE "billing_gathering_invoice_lines";
