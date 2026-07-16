-- modify "balance_snapshots" table
ALTER TABLE "balance_snapshots" ADD COLUMN "unit_config" jsonb NULL;
-- modify "entitlements" table
ALTER TABLE "entitlements" ADD COLUMN "unit_config" jsonb NULL;
