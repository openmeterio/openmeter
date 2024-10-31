-- reverse: create index "billinginvoicevalidationissue_namespace_invoice_id_dedupe_hash" to table: "billing_invoice_validation_issues"
DROP INDEX "billinginvoicevalidationissue_namespace_invoice_id_dedupe_hash";
-- reverse: create index "billinginvoicevalidationissue_namespace" to table: "billing_invoice_validation_issues"
DROP INDEX "billinginvoicevalidationissue_namespace";
-- reverse: create index "billinginvoicevalidationissue_id" to table: "billing_invoice_validation_issues"
DROP INDEX "billinginvoicevalidationissue_id";
-- reverse: create "billing_invoice_validation_issues" table
DROP TABLE "billing_invoice_validation_issues";
