-- reverse: modify "charge_usage_based" feature foreign key
ALTER TABLE "charge_usage_based" DROP CONSTRAINT "charge_usage_based_features_usage_based_charges", ADD CONSTRAINT "charge_usage_based_features_usage_based_charges" FOREIGN KEY ("feature_id") REFERENCES "features" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- reverse: modify "charge_usage_based" table
ALTER TABLE "charge_usage_based" ALTER COLUMN "feature_id" SET NOT NULL;
