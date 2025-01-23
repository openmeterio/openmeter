-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "payment_verification_needed" boolean NOT NULL DEFAULT false, ADD COLUMN "payment_verification_received" boolean NOT NULL DEFAULT false;
