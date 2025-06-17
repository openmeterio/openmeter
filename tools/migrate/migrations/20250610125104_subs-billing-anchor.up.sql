-- modify "subscriptions" table
ALTER TABLE "subscriptions" ADD COLUMN "billing_anchor" timestamptz;

-- update all entries to set the billing anchor to the active_from time
UPDATE "subscriptions" SET "billing_anchor" = "active_from" WHERE "billing_anchor" IS NULL;

-- make the billing anchor not nullable
ALTER TABLE "subscriptions" ALTER COLUMN "billing_anchor" SET NOT NULL;