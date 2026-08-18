-- modify "charge_flat_fee_runs" table
ALTER TABLE "charge_flat_fee_runs" ADD COLUMN "fiat_overage_credit_allocation_completed" boolean NOT NULL DEFAULT false;
-- modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "fiat_overage_credit_allocation_completed" boolean NOT NULL DEFAULT false;
