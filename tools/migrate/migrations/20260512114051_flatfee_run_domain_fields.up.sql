-- modify "charge_flat_fee_runs" table
ALTER TABLE "charge_flat_fee_runs" ADD COLUMN "line_id" character(26) NULL, ADD COLUMN "invoice_id" character(26) NULL, ADD COLUMN "no_fiat_transaction_required" boolean NULL;

UPDATE "charge_flat_fee_runs" AS "r"
SET "no_fiat_transaction_required" = true;

ALTER TABLE "charge_flat_fee_runs" ALTER COLUMN "no_fiat_transaction_required" SET NOT NULL;

-- modify "charge_flat_fee_run_invoiced_usages" table
ALTER TABLE "charge_flat_fee_run_invoiced_usages" DROP CONSTRAINT "charge_flat_fee_run_invoiced_usages_billing_invoice_lines_charg", DROP COLUMN "line_id", DROP COLUMN "mutable";
-- modify "charge_usage_based_run_invoiced_usages" table
ALTER TABLE "charge_usage_based_run_invoiced_usages" DROP COLUMN "line_id", DROP COLUMN "mutable";

ALTER TABLE "charge_flat_fee_runs" ADD CONSTRAINT "charge_flat_fee_runs_billing_invoice_lines_charge_flat_fee_runs" FOREIGN KEY ("line_id") REFERENCES "billing_invoice_lines" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_flat_fee_runs_billing_invoices_charge_flat_fee_runs" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "charge_flat_fee_runs_line_id_key" to table: "charge_flat_fee_runs"
CREATE UNIQUE INDEX "charge_flat_fee_runs_line_id_key" ON "charge_flat_fee_runs" ("line_id");
