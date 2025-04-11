-- modify "subscription_items" table
ALTER TABLE "subscription_items" ADD COLUMN "annotations" jsonb NULL;

-- We need to add an annotation to all existing subscription items
UPDATE "subscription_items" SET "annotations" = '{"subscription.owner": ["subscription"]}' WHERE "annotations" IS NULL;
