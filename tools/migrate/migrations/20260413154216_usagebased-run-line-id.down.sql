-- reverse: create index "charge_usage_based_runs_line_id_key" to table: "charge_usage_based_runs"
DROP INDEX "charge_usage_based_runs_line_id_key";
-- reverse: modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" DROP CONSTRAINT "charge_usage_based_runs_billing_invoice_lines_charge_usage_base", DROP COLUMN "line_id";
