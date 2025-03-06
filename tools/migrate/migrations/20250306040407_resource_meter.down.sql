-- reverse "meters" table
ALTER TABLE "meters" RENAME COLUMN "namespace" TO "organization_meters";
ALTER TABLE "meters" RENAME COLUMN "key" TO "slug";
ALTER TABLE "meters"
    DROP COLUMN "name",
    DROP COLUMN "event_from",
    DROP COLUMN "metadata",
    DROP COLUMN "deleted_at",
    ALTER COLUMN "value_property" SET NOT NULL,
    ALTER COLUMN "group_by" SET NOT NULL,
    ALTER COLUMN "description" SET DEFAULT ''::character varying,
    ADD COLUMN "window_size" character varying NOT NULL;

-- reverse dropping index "meter_slug_organization_meters"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "meter_slug_organization_meters" ON "meters" ("slug", "organization_meters");

-- reverse create index "meter_namespace"
DROP INDEX IF EXISTS "meter_namespace";
-- reverse create index "meter_namespace_id"
DROP INDEX IF EXISTS "meter_namespace_id";
-- reverse create index "meter_namespace_key_deleted_at"
DROP INDEX IF EXISTS "meter_namespace_key_deleted_at";
