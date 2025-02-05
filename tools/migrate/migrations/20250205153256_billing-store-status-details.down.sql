-- reverse: create index "billinginvoice_status_details_cache" to table: "billing_invoices"
DROP INDEX "billinginvoice_status_details_cache";
-- reverse: create index "billinginvoice_namespace_status" to table: "billing_invoices"
DROP INDEX "billinginvoice_namespace_status";
-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" DROP COLUMN "status_details_cache";
