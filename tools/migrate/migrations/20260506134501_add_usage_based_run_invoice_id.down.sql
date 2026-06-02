-- reverse: modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" DROP CONSTRAINT "charge_usage_based_runs_billing_invoices_charge_usage_based_run", DROP COLUMN "invoice_id";
