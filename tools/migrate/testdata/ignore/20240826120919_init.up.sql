-- create "features" table
CREATE TABLE "features" (
  "id" character(26) NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "metadata" jsonb NULL,
  "namespace" character varying NOT NULL,
  "name" character varying NOT NULL,
  "key" character varying NOT NULL,
  "meter_slug" character varying NULL,
  "meter_group_by_filters" jsonb NULL,
  "archived_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- create index "feature_id" to table: "features"
CREATE INDEX "feature_id" ON "features" ("id");
-- create index "feature_namespace_id" to table: "features"
CREATE INDEX "feature_namespace_id" ON "features" ("namespace", "id");
