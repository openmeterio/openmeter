-- create "meters" table
CREATE TABLE IF NOT EXISTS "meters" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "metadata" jsonb NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying NOT NULL,
  "description" character varying NULL,
  "key" character varying NOT NULL,
  "event_type" character varying NOT NULL,
  "value_property" character varying NULL,
  "group_by" jsonb NULL,
  "aggregation" character varying NOT NULL,
  "event_from" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- create index "meter_id" to table: "meters"
CREATE UNIQUE INDEX IF NOT EXISTS "meter_id" ON "meters" ("id");
-- create index "meter_namespace" to table: "meters"
CREATE INDEX IF NOT EXISTS "meter_namespace" ON "meters" ("namespace");
-- create index "meter_namespace_id" to table: "meters"
CREATE UNIQUE INDEX IF NOT EXISTS "meter_namespace_id" ON "meters" ("namespace", "id");
-- create index "meter_namespace_key_deleted_at" to table: "meters"
CREATE UNIQUE INDEX IF NOT EXISTS "meter_namespace_key_deleted_at" ON "meters" ("namespace", "key", "deleted_at");
