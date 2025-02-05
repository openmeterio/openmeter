-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "status_details_cache" jsonb NULL;
-- create index "billinginvoice_namespace_status" to table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_status" ON "billing_invoices" ("namespace", "status");
-- create index "billinginvoice_status_details_cache" to table: "billing_invoices"
CREATE INDEX "billinginvoice_status_details_cache" ON "billing_invoices" USING gin ("status_details_cache");
