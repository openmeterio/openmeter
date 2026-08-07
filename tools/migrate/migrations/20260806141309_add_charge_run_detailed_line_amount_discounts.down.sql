-- reverse: modify "charge_usage_based_run_detailed_line" table
ALTER TABLE "charge_usage_based_run_detailed_line" DROP COLUMN "amount_discounts";
-- reverse: modify "charge_flat_fee_run_detailed_lines" table
ALTER TABLE "charge_flat_fee_run_detailed_lines" DROP COLUMN "amount_discounts";
