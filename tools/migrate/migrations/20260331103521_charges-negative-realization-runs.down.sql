-- reverse: modify "charge_usage_based_run_credit_allocations" table
ALTER TABLE "charge_usage_based_run_credit_allocations" DROP CONSTRAINT "charge_ub_run_credit_alloc_run", DROP CONSTRAINT "charge_usage_based_run_credit_allocations_charge_usage_based_ru", ADD CONSTRAINT "charge_usage_based_run_credit_allocations_charge_usage_based_ru" FOREIGN KEY ("run_id") REFERENCES "charge_usage_based_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- reverse: modify "charge_flat_fee_credit_allocations" table
ALTER TABLE "charge_flat_fee_credit_allocations" DROP CONSTRAINT "charge_flat_fee_credit_allocations_charge_flat_fee_credit_alloc", DROP CONSTRAINT "charge_ff_credit_alloc_flat_fee", ADD CONSTRAINT "charge_flat_fee_credit_allocations_charge_flat_fees_credit_allo" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;

-- reverse: modify "charge_usage_based_run_credit_allocations" table
ALTER TABLE "charge_usage_based_run_credit_allocations" DROP COLUMN "corrects_realization_id", DROP COLUMN "type";
-- reverse: modify "charge_flat_fee_credit_allocations" table
ALTER TABLE "charge_flat_fee_credit_allocations" DROP COLUMN "corrects_realization_id", DROP COLUMN "type";
