-- reverse: create index "planratecard_phase_id_key" to table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_key";
-- reverse: create index "planratecard_phase_id_feature_key" to table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_feature_key";
-- reverse: create index "planratecard_namespace_key_deleted_at" to table: "plan_rate_cards"
DROP INDEX "planratecard_namespace_key_deleted_at";
-- reverse: create index "planratecard_namespace_id" to table: "plan_rate_cards"
DROP INDEX "planratecard_namespace_id";
-- reverse: create index "planratecard_namespace" to table: "plan_rate_cards"
DROP INDEX "planratecard_namespace";
-- reverse: create index "planratecard_id" to table: "plan_rate_cards"
DROP INDEX "planratecard_id";
-- reverse: create "plan_rate_cards" table
DROP TABLE "plan_rate_cards";
-- reverse: create index "planphase_plan_id_key" to table: "plan_phases"
DROP INDEX "planphase_plan_id_key";
-- reverse: create index "planphase_namespace_key_deleted_at" to table: "plan_phases"
DROP INDEX "planphase_namespace_key_deleted_at";
-- reverse: create index "planphase_namespace_key" to table: "plan_phases"
DROP INDEX "planphase_namespace_key";
-- reverse: create index "planphase_namespace_id" to table: "plan_phases"
DROP INDEX "planphase_namespace_id";
-- reverse: create index "planphase_namespace" to table: "plan_phases"
DROP INDEX "planphase_namespace";
-- reverse: create index "planphase_id" to table: "plan_phases"
DROP INDEX "planphase_id";
-- reverse: create "plan_phases" table
DROP TABLE "plan_phases";
-- reverse: create index "plan_namespace_key_version" to table: "plans"
DROP INDEX "plan_namespace_key_version";
-- reverse: create index "plan_namespace_key_deleted_at" to table: "plans"
DROP INDEX "plan_namespace_key_deleted_at";
-- reverse: create index "plan_namespace_id" to table: "plans"
DROP INDEX "plan_namespace_id";
-- reverse: create index "plan_namespace" to table: "plans"
DROP INDEX "plan_namespace";
-- reverse: create index "plan_id" to table: "plans"
DROP INDEX "plan_id";
-- reverse: create "plans" table
DROP TABLE "plans";
