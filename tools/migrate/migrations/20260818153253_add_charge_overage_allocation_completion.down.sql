-- reverse: modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" DROP COLUMN "fiat_overage_credit_allocation_completed";
-- reverse: modify "charge_flat_fee_runs" table
ALTER TABLE "charge_flat_fee_runs" DROP COLUMN "fiat_overage_credit_allocation_completed";
