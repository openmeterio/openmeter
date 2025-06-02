-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "billing_anchor" timestamptz NOT NULL;
