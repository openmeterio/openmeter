-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "payment_processing_entered_at" timestamptz NULL;
