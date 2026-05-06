-- modify "charge_flat_fee_detailed_line" table
ALTER TABLE "charge_flat_fee_detailed_line" ADD COLUMN "pricer_reference_id" character varying;
UPDATE "charge_flat_fee_detailed_line" SET "pricer_reference_id" = "child_unique_reference_id" WHERE "pricer_reference_id" IS NULL;
ALTER TABLE "charge_flat_fee_detailed_line" ALTER COLUMN "pricer_reference_id" SET NOT NULL;
-- modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ADD COLUMN "rating_engine" character varying;
UPDATE "charge_usage_based" SET "rating_engine" = 'delta' WHERE "rating_engine" IS NULL;
ALTER TABLE "charge_usage_based" ALTER COLUMN "rating_engine" SET NOT NULL;
-- modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "no_fiat_transaction_required" boolean NOT NULL DEFAULT false;
UPDATE "charge_usage_based_runs" AS r SET "no_fiat_transaction_required" = true FROM "charge_usage_based" AS c WHERE r."charge_id" = c."id" AND c."settlement_mode" = 'credit_only';
ALTER TABLE "charge_usage_based_runs" ALTER COLUMN "no_fiat_transaction_required" DROP DEFAULT;
-- modify "charge_usage_based_run_detailed_line" table
ALTER TABLE "charge_usage_based_run_detailed_line" ADD COLUMN "pricer_reference_id" character varying, ADD COLUMN "corrects_run_id" character(26) NULL, ADD CONSTRAINT "cub_run_corrected_detailed_lines" FOREIGN KEY ("corrects_run_id") REFERENCES "charge_usage_based_runs" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
UPDATE "charge_usage_based_run_detailed_line" SET "pricer_reference_id" = "child_unique_reference_id" WHERE "pricer_reference_id" IS NULL;
ALTER TABLE "charge_usage_based_run_detailed_line" ALTER COLUMN "pricer_reference_id" SET NOT NULL;
