-- modify "meters" table
-- note that alter table doesn't accept multiple column changes with renaming
-- atlas:nolint BC102
ALTER TABLE "meters" RENAME COLUMN "organization_meters" TO "namespace";
-- atlas:nolint BC102
ALTER TABLE "meters" RENAME COLUMN "slug" TO "key";

-- atlas:nolint DS103 BC102
ALTER TABLE "meters"
    ADD COLUMN "name" character varying NULL,
    ADD COLUMN "event_from" timestamptz NULL,
    ADD COLUMN "metadata" jsonb NULL,
    ADD COLUMN "deleted_at" timestamptz NULL,
    ALTER COLUMN "value_property" DROP NOT NULL,
    ALTER COLUMN "group_by" DROP NOT NULL,
    ALTER COLUMN "description" DROP DEFAULT,
    DROP COLUMN "window_size";

-- drop index "meter_slug_organization_meters" from table: "meters"
DROP INDEX IF EXISTS "meter_slug_organization_meters";

-- create index "meter_namespace" to table: "meters"
CREATE INDEX "meter_namespace" ON "meters" ("namespace");
-- create index "meter_namespace_id" to table: "meters"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "meter_namespace_id" ON "meters" ("namespace", "id");
-- create index "meter_namespace_key_deleted_at" to table: "meters"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "meter_namespace_key_deleted_at" ON "meters" ("namespace", "key", "deleted_at");

-- Update event_from column with the parsed timestamp from the id
-- This is a one time migraton to avoid picking up events before the meter was created.
UPDATE meters SET event_from = created_at;

-- Update name column with the slug to avoid null values
UPDATE meters SET name = key;

-- Add not null constraint to the columns
ALTER TABLE "meters" ALTER COLUMN "name" SET NOT NULL;
