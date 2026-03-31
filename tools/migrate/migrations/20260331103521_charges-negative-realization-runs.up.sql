-- modify "charge_flat_fee_credit_allocations" table
-- atlas:nolint MF103
ALTER TABLE "charge_flat_fee_credit_allocations" ADD COLUMN "type" character varying NOT NULL, ADD COLUMN "corrects_realization_id" character(26) NULL;
-- modify "charge_usage_based_run_credit_allocations" table
-- atlas:nolint MF103
ALTER TABLE "charge_usage_based_run_credit_allocations" ADD COLUMN "type" character varying NOT NULL, ADD COLUMN "corrects_realization_id" character(26) NULL;
