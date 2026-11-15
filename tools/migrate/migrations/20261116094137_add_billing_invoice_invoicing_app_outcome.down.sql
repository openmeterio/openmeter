-- reverse: modify "billing_invoices" table
ALTER TABLE "billing_invoices" DROP COLUMN "invoicing_app_outcome_reason", DROP COLUMN "invoicing_app_outcome";
