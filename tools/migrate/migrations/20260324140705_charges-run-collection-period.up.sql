-- modify "charge_usage_based_runs" table
-- atlas:nolint MF103
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "collection_end" timestamptz NOT NULL;
