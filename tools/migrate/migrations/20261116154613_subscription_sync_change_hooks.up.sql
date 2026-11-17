BEGIN;

-- modify "subscription_sync_states_v2" table
ALTER TABLE "subscription_sync_states_v2" ALTER COLUMN "last_success_at" DROP NOT NULL;

-- A downgrade fills this column for the old schema. New still means no recorded
-- sync result, so remove that placeholder when upgrading again.
UPDATE "subscription_sync_states_v2"
SET "last_success_at" = NULL
WHERE "last_success_state" = 'new' AND "last_success_at" IS NOT NULL;

COMMIT;
