-- reverse: create index "planaddon_namespace_plan_id_addon_id" to table: "plan_addons"
DROP INDEX "planaddon_namespace_plan_id_addon_id";
-- reverse: create index "planaddon_namespace" to table: "plan_addons"
DROP INDEX "planaddon_namespace";
-- reverse: create index "planaddon_id" to table: "plan_addons"
DROP INDEX "planaddon_id";
-- reverse: create index "planaddon_annotations" to table: "plan_addons"
DROP INDEX "planaddon_annotations";
-- reverse: create "plan_addons" table
DROP TABLE "plan_addons";
