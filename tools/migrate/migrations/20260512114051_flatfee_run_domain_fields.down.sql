-- reverse: create index "charge_flat_fee_runs_line_id_key" to table: "charge_flat_fee_runs"
DROP INDEX "charge_flat_fee_runs_line_id_key";
-- reverse: modify "charge_flat_fee_runs" table
ALTER TABLE "charge_flat_fee_runs" DROP CONSTRAINT "charge_flat_fee_runs_billing_invoices_charge_flat_fee_runs", DROP CONSTRAINT "charge_flat_fee_runs_billing_invoice_lines_charge_flat_fee_runs";
-- reverse: modify "charge_usage_based_run_invoiced_usages" table
ALTER TABLE "charge_usage_based_run_invoiced_usages" ADD COLUMN "line_id" character(26) NULL, ADD COLUMN "mutable" boolean NOT NULL DEFAULT false;
ALTER TABLE "charge_usage_based_run_invoiced_usages" ALTER COLUMN "mutable" DROP DEFAULT;
-- reverse: modify "charge_flat_fee_run_invoiced_usages" table
ALTER TABLE "charge_flat_fee_run_invoiced_usages" ADD COLUMN "line_id" character(26) NULL, ADD COLUMN "mutable" boolean NOT NULL DEFAULT false;
ALTER TABLE "charge_flat_fee_run_invoiced_usages" ALTER COLUMN "mutable" DROP DEFAULT;
ALTER TABLE "charge_flat_fee_run_invoiced_usages" ADD CONSTRAINT "charge_flat_fee_run_invoiced_usages_billing_invoice_lines_charg" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- reverse: modify "charge_flat_fee_runs" table
ALTER TABLE "charge_flat_fee_runs" DROP COLUMN "no_fiat_transaction_required", DROP COLUMN "invoice_id", DROP COLUMN "line_id";
