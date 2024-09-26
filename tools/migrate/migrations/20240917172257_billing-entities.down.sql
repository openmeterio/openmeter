-- reverse: create index "billinginvoiceitem_namespace_invoice_id" to table: "billing_invoice_items"
DROP INDEX "billinginvoiceitem_namespace_invoice_id";
-- reverse: create index "billinginvoiceitem_namespace_id" to table: "billing_invoice_items"
DROP INDEX "billinginvoiceitem_namespace_id";
-- reverse: create index "billinginvoiceitem_namespace_customer_id" to table: "billing_invoice_items"
DROP INDEX "billinginvoiceitem_namespace_customer_id";
-- reverse: create index "billinginvoiceitem_id" to table: "billing_invoice_items"
DROP INDEX "billinginvoiceitem_id";
-- reverse: create "billing_invoice_items" table
DROP TABLE "billing_invoice_items";
-- reverse: create index "billinginvoice_namespace_status" to table: "billing_invoices"
DROP INDEX "billinginvoice_namespace_status";
-- reverse: create index "billinginvoice_namespace_id" to table: "billing_invoices"
DROP INDEX "billinginvoice_namespace_id";
-- reverse: create index "billinginvoice_namespace_due_date" to table: "billing_invoices"
DROP INDEX "billinginvoice_namespace_due_date";
-- reverse: create index "billinginvoice_namespace_customer_id" to table: "billing_invoices"
DROP INDEX "billinginvoice_namespace_customer_id";
-- reverse: create index "billinginvoice_id" to table: "billing_invoices"
DROP INDEX "billinginvoice_id";
-- reverse: create "billing_invoices" table
DROP TABLE "billing_invoices";
-- reverse: create index "billingprofile_namespace_key" to table: "billing_profiles"
DROP INDEX IF EXISTS "billingprofile_namespace_key";
-- reverse: create index "billingprofile_namespace_id" to table: "billing_profiles"
DROP INDEX "billingprofile_namespace_id";
-- reverse: create index "billingprofile_namespace_default" to table: "billing_profiles"
DROP INDEX "billingprofile_namespace_default";
-- reverse: create index "billingprofile_id" to table: "billing_profiles"
DROP INDEX "billingprofile_id";
-- reverse: create "billing_profiles" table
DROP TABLE "billing_profiles";
-- reverse: create index "billingworkflowconfig_namespace_id" to table: "billing_workflow_configs"
DROP INDEX "billingworkflowconfig_namespace_id";
-- reverse: create index "billingworkflowconfig_id" to table: "billing_workflow_configs"
DROP INDEX "billingworkflowconfig_id";
-- reverse: create "billing_workflow_configs" table
DROP TABLE "billing_workflow_configs";
