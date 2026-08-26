-- reverse: modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" DROP CONSTRAINT "charge_ub_run_prior_run", DROP COLUMN "prior_run_id", DROP COLUMN "schema_level";
