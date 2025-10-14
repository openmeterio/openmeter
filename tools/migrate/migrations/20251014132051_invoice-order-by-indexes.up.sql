-- create index "billinginvoice_namespace_created_at" to table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_created_at" ON "billing_invoices" ("namespace", "created_at");
-- create index "billinginvoice_namespace_issued_at" to table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_issued_at" ON "billing_invoices" ("namespace", "issued_at");
-- create index "billinginvoice_namespace_period_start" to table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_period_start" ON "billing_invoices" ("namespace", "period_start");
-- create index "billinginvoice_namespace_updated_at" to table: "billing_invoices"
CREATE INDEX "billinginvoice_namespace_updated_at" ON "billing_invoices" ("namespace", "updated_at");
