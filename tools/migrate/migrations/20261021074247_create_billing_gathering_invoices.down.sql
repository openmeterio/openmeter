-- reverse: create "billing_invoice_search_v1s" view
DROP VIEW "billing_invoice_search_v1s";
-- reverse: create index "billinggatheringinvoiceline_invoice_id" to table: "billing_gathering_invoice_lines"
DROP INDEX "billinggatheringinvoiceline_invoice_id";
-- reverse: modify "billing_gathering_invoice_lines" table
ALTER TABLE "billing_gathering_invoice_lines" DROP CONSTRAINT "billing_gathering_line_invoice_fk", DROP CONSTRAINT "service_period_not_inverted", ADD CONSTRAINT "billing_gathering_line_invoice_fk" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- reverse: create index "billinggatheringinvoice_namespace_updated_at" to table: "billing_gathering_invoices"
DROP INDEX "billinggatheringinvoice_namespace_updated_at";
-- reverse: create index "billinggatheringinvoice_namespace_next_collection_at" to table: "billing_gathering_invoices"
DROP INDEX "billinggatheringinvoice_namespace_next_collection_at";
-- reverse: create index "billinggatheringinvoice_namespace_id" to table: "billing_gathering_invoices"
DROP INDEX "billinggatheringinvoice_namespace_id";
-- reverse: create index "billinggatheringinvoice_namespace_customer_id_currency" to table: "billing_gathering_invoices"
DROP INDEX "billinggatheringinvoice_namespace_customer_id_currency";
-- reverse: create index "billinggatheringinvoice_namespace_customer_id" to table: "billing_gathering_invoices"
DROP INDEX "billinggatheringinvoice_namespace_customer_id";
-- reverse: create index "billinggatheringinvoice_namespace_created_at" to table: "billing_gathering_invoices"
DROP INDEX "billinggatheringinvoice_namespace_created_at";
-- reverse: create index "billinggatheringinvoice_namespace" to table: "billing_gathering_invoices"
DROP INDEX "billinggatheringinvoice_namespace";
-- reverse: create index "billinggatheringinvoice_id" to table: "billing_gathering_invoices"
DROP INDEX "billinggatheringinvoice_id";
-- reverse: create index "billinggatheringinvoice_customer_id" to table: "billing_gathering_invoices"
DROP INDEX "billinggatheringinvoice_customer_id";
-- reverse: create "billing_gathering_invoices" table
DROP TABLE "billing_gathering_invoices";
