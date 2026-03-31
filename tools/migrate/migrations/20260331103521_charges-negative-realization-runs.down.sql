-- reverse: modify "charge_usage_based_run_credit_allocations" table
ALTER TABLE "charge_usage_based_run_credit_allocations" DROP COLUMN "corrects_realization_id", DROP COLUMN "type";
-- reverse: modify "charge_flat_fee_credit_allocations" table
ALTER TABLE "charge_flat_fee_credit_allocations" DROP COLUMN "corrects_realization_id", DROP COLUMN "type";
