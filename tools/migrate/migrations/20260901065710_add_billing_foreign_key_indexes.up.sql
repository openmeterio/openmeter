-- atlas:txmode none

-- create index "billing_invoice_line_discounts_line_id_idx" to table: "billing_invoice_line_discounts"
CREATE INDEX CONCURRENTLY "billing_invoice_line_discounts_line_id_idx" ON "billing_invoice_line_discounts" ("line_id");
-- create index "billing_invoice_line_usage_discounts_line_id_idx" to table: "billing_invoice_line_usage_discounts"
CREATE INDEX CONCURRENTLY "billing_invoice_line_usage_discounts_line_id_idx" ON "billing_invoice_line_usage_discounts" ("line_id");
-- create index "billing_invoice_lines_fee_config_id_idx" to table: "billing_invoice_lines"
CREATE INDEX CONCURRENTLY "billing_invoice_lines_fee_config_id_idx" ON "billing_invoice_lines" ("fee_line_config_id");
-- create index "billing_invoice_lines_parent_id_idx" to table: "billing_invoice_lines"
CREATE INDEX CONCURRENTLY "billing_invoice_lines_parent_id_idx" ON "billing_invoice_lines" ("parent_line_id");
-- create index "billing_invoice_lines_usage_config_id_idx" to table: "billing_invoice_lines"
CREATE INDEX CONCURRENTLY "billing_invoice_lines_usage_config_id_idx" ON "billing_invoice_lines" ("usage_based_line_config_id");
-- create index "billing_std_invoice_detail_amount_discounts_line_id_idx" to table: "billing_standard_invoice_detailed_line_amount_discounts"
CREATE INDEX CONCURRENTLY "billing_std_invoice_detail_amount_discounts_line_id_idx" ON "billing_standard_invoice_detailed_line_amount_discounts" ("line_id");
-- create index "billing_std_invoice_detailed_lines_parent_id_idx" to table: "billing_standard_invoice_detailed_lines"
CREATE INDEX CONCURRENTLY "billing_std_invoice_detailed_lines_parent_id_idx" ON "billing_standard_invoice_detailed_lines" ("parent_line_id");
