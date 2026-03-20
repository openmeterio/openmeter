-- modify "features" table
ALTER TABLE "features" ADD COLUMN "meter_id" character(26) NULL, ADD CONSTRAINT "features_meters_feature" FOREIGN KEY ("meter_id") REFERENCES "meters" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "feature_namespace_meter_id" to table: "features"
CREATE INDEX "feature_namespace_meter_id" ON "features" ("namespace", "meter_id") WHERE (archived_at IS NULL);
-- populate meter_id from meter_slug by joining on key + namespace
-- includes soft-deleted meters so existing references are preserved
UPDATE features f
SET meter_id = m.id
FROM meters m
WHERE f.meter_slug = m.key
  AND f.namespace = m.namespace
  AND f.meter_slug IS NOT NULL
  AND f.meter_id IS NULL;
