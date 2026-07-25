-- reverse: modify "charge_usage_based_overrides" table
ALTER TABLE "charge_usage_based_overrides" ALTER COLUMN "feature_key" SET NOT NULL;
