-- reverse: modify "entitlements" table
ALTER TABLE "entitlements" DROP COLUMN "unit_config";
-- reverse: modify "balance_snapshots" table
ALTER TABLE "balance_snapshots" DROP COLUMN "unit_config";
