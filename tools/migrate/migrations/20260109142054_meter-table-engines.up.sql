-- create "meter_table_engines" table
CREATE TABLE "meter_table_engines" (
  "id" character(26) NOT NULL,
  "namespace" character varying NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  "engine" character varying NOT NULL DEFAULT 'events',
  "status" character varying NOT NULL DEFAULT 'preparing',
  "state" jsonb NOT NULL,
  "meter_id" character(26) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "meter_table_engines_meters_table_engine" FOREIGN KEY ("meter_id") REFERENCES "meters" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "meter_table_engines_meter_id_key" to table: "meter_table_engines"
CREATE UNIQUE INDEX "meter_table_engines_meter_id_key" ON "meter_table_engines" ("meter_id");
-- create index "metertableengine_id" to table: "meter_table_engines"
CREATE UNIQUE INDEX "metertableengine_id" ON "meter_table_engines" ("id");
-- create index "metertableengine_meter_id" to table: "meter_table_engines"
CREATE UNIQUE INDEX "metertableengine_meter_id" ON "meter_table_engines" ("meter_id");
-- create index "metertableengine_namespace" to table: "meter_table_engines"
CREATE INDEX "metertableengine_namespace" ON "meter_table_engines" ("namespace");
