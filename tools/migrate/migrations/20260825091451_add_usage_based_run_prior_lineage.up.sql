-- modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "schema_level" smallint NOT NULL DEFAULT 1, ADD COLUMN "prior_run_id" character(26) NULL, ADD CONSTRAINT "charge_ub_run_prior_run" FOREIGN KEY ("prior_run_id") REFERENCES "charge_usage_based_runs" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
