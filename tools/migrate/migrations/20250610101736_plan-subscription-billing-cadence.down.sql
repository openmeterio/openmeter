-- reverse: modify "subscriptions" table
ALTER TABLE "subscriptions" DROP COLUMN "pro_rating_config", DROP COLUMN "billing_cadence";
-- reverse: modify "plans" table
ALTER TABLE "plans" DROP COLUMN "pro_rating_config", DROP COLUMN "billing_cadence";
