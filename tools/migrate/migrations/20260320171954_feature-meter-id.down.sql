-- reverse: create index "feature_namespace_meter_id" to table: "features"
DROP INDEX "feature_namespace_meter_id";
-- reverse: modify "features" table
ALTER TABLE "features" DROP CONSTRAINT "features_meters_feature", DROP COLUMN "meter_id";
