-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "draft_until" timestamptz NULL;
