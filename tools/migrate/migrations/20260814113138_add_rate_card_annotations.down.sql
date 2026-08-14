BEGIN;

-- reverse: modify "plan_rate_cards" table
ALTER TABLE "plan_rate_cards" DROP COLUMN "annotations";
-- reverse: modify "addon_rate_cards" table
ALTER TABLE "addon_rate_cards" DROP COLUMN "annotations";

COMMIT;
