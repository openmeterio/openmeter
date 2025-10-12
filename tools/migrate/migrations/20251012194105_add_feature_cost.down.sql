-- reverse: modify "features" table
ALTER TABLE "features" DROP COLUMN "cost_provider_id", DROP COLUMN "cost_unit_amount", DROP COLUMN "cost_currency", DROP COLUMN "cost_kind";
