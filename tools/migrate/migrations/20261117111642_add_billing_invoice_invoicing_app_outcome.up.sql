-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "invoicing_app_outcome" character varying NULL, ADD COLUMN "invoicing_app_outcome_reason" character varying NULL;
