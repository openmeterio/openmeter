-- modify "features" table
ALTER TABLE "features" ADD COLUMN "meter_id" character(26) NULL, ADD CONSTRAINT "features_meters_feature" FOREIGN KEY ("meter_id") REFERENCES "meters" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- create index "feature_namespace_meter_id" to table: "features"
CREATE INDEX "feature_namespace_meter_id" ON "features" ("namespace", "meter_id") WHERE (archived_at IS NULL);
-- populate meter_id from meter_slug using a deterministic match
-- prefer the active meter for a key, otherwise fall back to the most recently deleted one
WITH resolved_meters AS (
    SELECT DISTINCT ON (namespace, key)
        id,
        namespace,
        key
    FROM meters
    ORDER BY namespace, key, (deleted_at IS NULL) DESC, deleted_at DESC NULLS LAST, updated_at DESC, created_at DESC, id DESC
)
UPDATE features f
SET meter_id = m.id
FROM resolved_meters m
WHERE f.meter_slug = m.key
  AND f.namespace = m.namespace
  AND f.meter_slug IS NOT NULL
  AND f.meter_id IS NULL;
