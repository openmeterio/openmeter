-- reverse: modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" DROP COLUMN "no_fiat_transaction_required";
-- reverse: modify "charge_usage_based_run_detailed_line" table
ALTER TABLE "charge_usage_based_run_detailed_line" DROP COLUMN "pricer_reference_id";
-- reverse: modify "charge_flat_fee_detailed_line" table
ALTER TABLE "charge_flat_fee_detailed_line" DROP COLUMN "pricer_reference_id";
-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP COLUMN "rating_engine";
-- reverse: modify "charge_usage_based_run_detailed_line" table
ALTER TABLE "charge_usage_based_run_detailed_line" DROP CONSTRAINT "cub_run_corrected_detailed_lines", DROP COLUMN "corrects_run_id";
