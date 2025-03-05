-- reverse: create index "meter_namespace_key_deleted_at" to table: "meters"
DROP INDEX "meter_namespace_key_deleted_at";
-- reverse: create index "meter_namespace_id" to table: "meters"
DROP INDEX "meter_namespace_id";
-- reverse: create index "meter_namespace" to table: "meters"
DROP INDEX "meter_namespace";
-- reverse: create index "meter_id" to table: "meters"
DROP INDEX "meter_id";
-- reverse: create "meters" table
DROP TABLE "meters";
