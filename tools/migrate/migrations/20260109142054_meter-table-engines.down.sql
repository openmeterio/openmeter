-- reverse: create index "metertableengine_namespace" to table: "meter_table_engines"
DROP INDEX "metertableengine_namespace";
-- reverse: create index "metertableengine_meter_id" to table: "meter_table_engines"
DROP INDEX "metertableengine_meter_id";
-- reverse: create index "metertableengine_id" to table: "meter_table_engines"
DROP INDEX "metertableengine_id";
-- reverse: create index "meter_table_engines_meter_id_key" to table: "meter_table_engines"
DROP INDEX "meter_table_engines_meter_id_key";
-- reverse: create "meter_table_engines" table
DROP TABLE "meter_table_engines";
