-- add initial realization run type and backfill it from the current type
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "initial_type" character varying NULL;
UPDATE "charge_usage_based_runs" SET "initial_type" = "type" WHERE "initial_type" IS NULL;
ALTER TABLE "charge_usage_based_runs" ALTER COLUMN "initial_type" SET NOT NULL;
