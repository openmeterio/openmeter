-- modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "detailed_lines_include_credit_allocations" boolean NOT NULL DEFAULT false;
