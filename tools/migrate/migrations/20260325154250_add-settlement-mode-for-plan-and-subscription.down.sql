-- reverse: modify "subscriptions" table
ALTER TABLE "subscriptions" DROP COLUMN "settlement_mode";
-- reverse: modify "plans" table
ALTER TABLE "plans" DROP COLUMN "settlement_mode";
