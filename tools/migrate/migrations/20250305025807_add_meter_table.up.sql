-- create table "meters"
CREATE TABLE IF NOT EXISTS "meters" (
    "id" character(26) NOT NULL,
    "created_at" timestamp with time zone NOT NULL,
    "updated_at" timestamp with time zone NOT NULL,
    "slug" character varying NOT NULL,
    "description" character varying DEFAULT ''::character varying,
    "event_type" character varying NOT NULL,
    "value_property" character varying NOT NULL,
    "group_by" jsonb NOT NULL,
    "aggregation" character varying NOT NULL,
    "window_size" character varying NOT NULL,
    "organization_meters" character varying NOT NULL,
    PRIMARY KEY ("id")
);

-- modify "meters" table
-- note that alter table doesn't accept multiple column changes with renaming
ALTER TABLE "meters" RENAME COLUMN "organization_meters" TO "namespace";
ALTER TABLE "meters" RENAME COLUMN "slug" TO "key";
ALTER TABLE "meters"
    ADD COLUMN "name" character varying NULL,
    ADD COLUMN "event_from" timestamptz NULL,
    ADD COLUMN "metadata" jsonb NULL,
    ADD COLUMN "deleted_at" timestamptz NULL,
    ALTER COLUMN "value_property" DROP NOT NULL,
    ALTER COLUMN "group_by" DROP NOT NULL,
    DROP COLUMN "window_size";

-- drop index "meter_slug_organization_meters" from table: "meters"
DROP INDEX IF EXISTS "meter_slug_organization_meters";
-- create index "meter_namespace" to table: "meters"
CREATE INDEX IF NOT EXISTS "meter_namespace" ON "meters" ("namespace");
-- create index "meter_namespace_id" to table: "meters"
-- atlas:nolint MF101s
CREATE UNIQUE INDEX IF NOT EXISTS "meter_namespace_id" ON "meters" ("namespace", "id");
-- create index "meter_namespace_key_deleted_at" to table: "meters"
-- atlas:nolint MF101
CREATE UNIQUE INDEX IF NOT EXISTS "meter_namespace_key_deleted_at" ON "meters" ("namespace", "key", "deleted_at");

-- Update event_from column with the parsed timestamp from the id
-- This is a one time migraton to avoid picking up events before the meter was created.
UPDATE meters SET event_from = created_at;

-- Update name column with the slug to avoid null values
UPDATE meters SET name = key;

-- Add not null constraint to the columns
ALTER TABLE "meters" ALTER COLUMN "name" SET NOT NULL;
