-- reverse: modify "entitlements" table
ALTER TABLE "entitlements" DROP COLUMN "balance_snapshot_invalidation_version";
