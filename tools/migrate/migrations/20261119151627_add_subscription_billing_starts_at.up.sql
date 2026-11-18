-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "billing_starts_at" timestamptz NULL;
