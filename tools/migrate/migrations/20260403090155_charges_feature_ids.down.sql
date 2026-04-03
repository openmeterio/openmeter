-- reverse: modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" DROP CONSTRAINT "charge_usage_based_runs_features_usage_based_runs", DROP COLUMN "feature_id";
-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_features_usage_based_charges", DROP COLUMN "feature_id";
-- reverse: modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" DROP CONSTRAINT "charge_flat_fees_features_flat_fee_charges", DROP COLUMN "feature_id";
