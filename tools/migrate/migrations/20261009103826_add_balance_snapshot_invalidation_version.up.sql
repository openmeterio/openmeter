-- modify "entitlements" table
ALTER TABLE "entitlements" ADD COLUMN "balance_snapshot_invalidation_version" bigint NOT NULL DEFAULT 0;
