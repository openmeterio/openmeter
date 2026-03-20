-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_charge_usage_based_runs_current_run", DROP COLUMN "current_realization_run_id", DROP COLUMN "status";
-- reverse: modify "charges" table
ALTER TABLE "charges" DROP COLUMN "advance_after";
-- reverse: modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" DROP COLUMN "total", DROP COLUMN "credits_total", DROP COLUMN "discounts_total", DROP COLUMN "charges_total", DROP COLUMN "taxes_exclusive_total", DROP COLUMN "taxes_inclusive_total", DROP COLUMN "taxes_total", DROP COLUMN "amount";
-- reverse: modify "charge_usage_based_run_credit_allocations" table
ALTER TABLE "charge_usage_based_run_credit_allocations" DROP COLUMN "sort_hint";
-- reverse: modify "charge_flat_fee_credit_allocations" table
ALTER TABLE "charge_flat_fee_credit_allocations" DROP COLUMN "sort_hint";
