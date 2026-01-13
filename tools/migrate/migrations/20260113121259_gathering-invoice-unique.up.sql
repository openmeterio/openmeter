-- create index "billinginvoice_namespace_customer_id_currency" to table: "billing_invoices"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billinginvoice_namespace_customer_id_currency" ON "billing_invoices" ("namespace", "customer_id", "currency") WHERE ((deleted_at IS NULL) AND ((status)::text = 'gathering'::text));
