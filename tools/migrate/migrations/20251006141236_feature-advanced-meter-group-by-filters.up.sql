-- modify "features" table
ALTER TABLE "features" ADD COLUMN "advanced_meter_group_by_filters" jsonb NULL;

-- Copy existing meter_group_by_filters data to advanced_meter_group_by_filters
-- Convert simple key-value pairs to FilterString objects with $eq operator
UPDATE "features"
SET "advanced_meter_group_by_filters" = (
    SELECT COALESCE(
        jsonb_object_agg(
            kv.key,
            jsonb_build_object('$eq', kv.value)
        ),
        '{}'::jsonb
    )
    FROM jsonb_each_text("meter_group_by_filters") AS kv(key, value)
)
WHERE "meter_group_by_filters" IS NOT NULL 
  AND "meter_group_by_filters" != 'null'::jsonb
  AND jsonb_typeof("meter_group_by_filters") = 'object';
