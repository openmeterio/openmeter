-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "deletion_source" character varying NULL;
