-- modify "charge_flat_fees" table
ALTER TABLE "charge_flat_fees" ADD COLUMN "feature_id" character(26) NULL, ADD CONSTRAINT "charge_flat_fees_features_flat_fee_charges" FOREIGN KEY ("feature_id") REFERENCES "features" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ADD COLUMN "feature_id" character(26) NOT NULL, ADD CONSTRAINT "charge_usage_based_features_usage_based_charges" FOREIGN KEY ("feature_id") REFERENCES "features" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "feature_id" character(26) NOT NULL, ADD CONSTRAINT "charge_usage_based_runs_features_usage_based_runs" FOREIGN KEY ("feature_id") REFERENCES "features" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
