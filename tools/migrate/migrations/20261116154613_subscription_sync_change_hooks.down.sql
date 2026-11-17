BEGIN;

-- Preserve new rows and their revisions while satisfying the old NOT NULL column.
UPDATE "subscription_sync_states_v2"
SET "last_success_at" = now()
WHERE "last_success_state" = 'new' AND "last_success_at" IS NULL;

-- reverse: modify "subscription_sync_states_v2" table
ALTER TABLE "subscription_sync_states_v2" ALTER COLUMN "last_success_at" SET NOT NULL;

COMMIT;
