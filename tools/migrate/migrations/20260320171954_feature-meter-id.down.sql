-- reverse: create index "feature_namespace_meter_id" to table: "features"
DROP INDEX "feature_namespace_meter_id";
-- repopulate meter_slug from meter_id before dropping the new foreign key column
UPDATE features f
SET meter_slug = m.key
FROM meters m
WHERE f.meter_id = m.id
  AND f.meter_id IS NOT NULL
  AND (f.meter_slug IS NULL OR f.meter_slug <> m.key);
-- reverse: modify "features" table
ALTER TABLE "features" DROP CONSTRAINT "features_meters_feature", DROP COLUMN "meter_id";
