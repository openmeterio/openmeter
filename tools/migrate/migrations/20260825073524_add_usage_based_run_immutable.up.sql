-- modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "immutable" boolean NOT NULL DEFAULT false;
