-- modify "charge_flat_fee_credit_allocations" table
-- atlas:nolint MF103
ALTER TABLE "charge_flat_fee_credit_allocations" ADD COLUMN "sort_hint" bigint NOT NULL;
-- modify "charge_usage_based_run_credit_allocations" table
-- atlas:nolint MF103
ALTER TABLE "charge_usage_based_run_credit_allocations" ADD COLUMN "sort_hint" bigint NOT NULL;
-- modify "charge_usage_based_runs" table
-- atlas:nolint MF103
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "amount" numeric NOT NULL, ADD COLUMN "taxes_total" numeric NOT NULL, ADD COLUMN "taxes_inclusive_total" numeric NOT NULL, ADD COLUMN "taxes_exclusive_total" numeric NOT NULL, ADD COLUMN "charges_total" numeric NOT NULL, ADD COLUMN "discounts_total" numeric NOT NULL, ADD COLUMN "credits_total" numeric NOT NULL, ADD COLUMN "total" numeric NOT NULL;
-- modify "charges" table
ALTER TABLE "charges" ADD COLUMN "advance_after" timestamptz NULL;
-- modify "charge_usage_based" table
-- atlas:nolint MF103
ALTER TABLE "charge_usage_based" ADD COLUMN "status" character varying NOT NULL, ADD COLUMN "current_realization_run_id" character(26) NULL, ADD CONSTRAINT "charge_usage_based_charge_usage_based_runs_current_run" FOREIGN KEY ("current_realization_run_id") REFERENCES "charge_usage_based_runs" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
