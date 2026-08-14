BEGIN;

-- modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" ADD COLUMN "annotations" jsonb NULL;
-- modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" ADD COLUMN "annotations" jsonb NULL;

COMMIT;
