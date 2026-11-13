-- create index "billinginvoiceline_namespace_charge_id" to table: "billing_invoice_lines"
CREATE INDEX CONCURRENTLY "billinginvoiceline_namespace_charge_id" ON "billing_invoice_lines" ("namespace", "charge_id");
