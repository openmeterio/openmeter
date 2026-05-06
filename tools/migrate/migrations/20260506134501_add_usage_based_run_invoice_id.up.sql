-- modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "invoice_id" character(26) NULL, ADD CONSTRAINT "charge_usage_based_runs_billing_invoices_charge_usage_based_run" FOREIGN KEY ("invoice_id") REFERENCES "billing_invoices" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
