-- modify "balance_snapshots" table
ALTER TABLE "balance_snapshots" ADD COLUMN "usage_snapshot" jsonb NULL;
