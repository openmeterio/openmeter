-- reverse: create index "subscription_annotations" to table: "subscriptions"
DROP INDEX "subscription_annotations";
-- reverse: modify "subscriptions" table
ALTER TABLE "subscriptions" DROP COLUMN "annotations";
