-- reverse: modify "subscriptions" table
ALTER TABLE "subscriptions" DROP COLUMN "payment_verification_received", DROP COLUMN "payment_verification_needed";
