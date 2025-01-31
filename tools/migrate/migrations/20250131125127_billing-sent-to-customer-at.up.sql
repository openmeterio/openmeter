-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ADD COLUMN "sent_to_customer_at" timestamptz NULL;
