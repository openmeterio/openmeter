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

-- create index "meter_id" to table: "meters"
-- atlas:nolint MF101
CREATE UNIQUE INDEX IF NOT EXISTS "meter_id" ON "meters" ("id");

