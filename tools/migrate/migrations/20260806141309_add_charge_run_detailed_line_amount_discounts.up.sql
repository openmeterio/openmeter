-- modify "charge_flat_fee_run_detailed_lines" table
ALTER TABLE "charge_flat_fee_run_detailed_lines" ADD COLUMN "amount_discounts" jsonb NULL;
-- modify "charge_usage_based_run_detailed_line" table
ALTER TABLE "charge_usage_based_run_detailed_line" ADD COLUMN "amount_discounts" jsonb NULL;
