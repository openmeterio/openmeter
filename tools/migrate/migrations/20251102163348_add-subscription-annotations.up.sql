-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "annotations" jsonb NULL;
-- create index "subscription_annotations" to table: "subscriptions"
CREATE INDEX "subscription_annotations" ON "subscriptions" USING gin ("annotations");
