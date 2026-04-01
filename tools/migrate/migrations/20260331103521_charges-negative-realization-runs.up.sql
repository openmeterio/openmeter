-- modify "charge_flat_fee_credit_allocations" table
-- atlas:nolint MF103
ALTER TABLE "charge_flat_fee_credit_allocations" ADD COLUMN "type" character varying NOT NULL, ADD COLUMN "corrects_realization_id" character(26) NULL;
-- modify "charge_usage_based_run_credit_allocations" table
-- atlas:nolint MF103
ALTER TABLE "charge_usage_based_run_credit_allocations" ADD COLUMN "type" character varying NOT NULL, ADD COLUMN "corrects_realization_id" character(26) NULL;

-- modify "charge_flat_fee_credit_allocations" table
-- atlas:nolint CD101
ALTER TABLE "charge_flat_fee_credit_allocations" DROP CONSTRAINT "charge_flat_fee_credit_allocations_charge_flat_fees_credit_allo", ADD CONSTRAINT "charge_ff_credit_alloc_flat_fee" FOREIGN KEY ("charge_id") REFERENCES "charge_flat_fees" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "charge_flat_fee_credit_allocations_charge_flat_fee_credit_alloc" FOREIGN KEY ("corrects_realization_id") REFERENCES "charge_flat_fee_credit_allocations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "charge_usage_based_run_credit_allocations" table
-- atlas:nolint CD101
ALTER TABLE "charge_usage_based_run_credit_allocations" DROP CONSTRAINT "charge_usage_based_run_credit_allocations_charge_usage_based_ru", ADD CONSTRAINT "charge_usage_based_run_credit_allocations_charge_usage_based_ru" FOREIGN KEY ("corrects_realization_id") REFERENCES "charge_usage_based_run_credit_allocations" ("id") ON UPDATE NO ACTION ON DELETE SET NULL, ADD CONSTRAINT "charge_ub_run_credit_alloc_run" FOREIGN KEY ("run_id") REFERENCES "charge_usage_based_runs" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
