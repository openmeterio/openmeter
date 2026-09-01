-- atlas:txmode none

-- reverse: create index "billing_std_invoice_detailed_lines_parent_id_idx" to table: "billing_standard_invoice_detailed_lines"
DROP INDEX CONCURRENTLY "billing_std_invoice_detailed_lines_parent_id_idx";
-- reverse: create index "billing_std_invoice_detail_amount_discounts_line_id_idx" to table: "billing_standard_invoice_detailed_line_amount_discounts"
DROP INDEX CONCURRENTLY "billing_std_invoice_detail_amount_discounts_line_id_idx";
-- reverse: create index "billing_invoice_lines_usage_config_id_idx" to table: "billing_invoice_lines"
DROP INDEX CONCURRENTLY "billing_invoice_lines_usage_config_id_idx";
-- reverse: create index "billing_invoice_lines_parent_id_idx" to table: "billing_invoice_lines"
DROP INDEX CONCURRENTLY "billing_invoice_lines_parent_id_idx";
-- reverse: create index "billing_invoice_lines_fee_config_id_idx" to table: "billing_invoice_lines"
DROP INDEX CONCURRENTLY "billing_invoice_lines_fee_config_id_idx";
-- reverse: create index "billing_invoice_line_usage_discounts_line_id_idx" to table: "billing_invoice_line_usage_discounts"
DROP INDEX CONCURRENTLY "billing_invoice_line_usage_discounts_line_id_idx";
-- reverse: create index "billing_invoice_line_discounts_line_id_idx" to table: "billing_invoice_line_discounts"
DROP INDEX CONCURRENTLY "billing_invoice_line_discounts_line_id_idx";
