-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "collection_at" timestamptz NULL;
