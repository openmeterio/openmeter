-- drop index "planphase_plan_id_index_deleted_at" from table: "plan_phases"
DROP INDEX "planphase_plan_id_index_deleted_at";
-- drop index "planphase_plan_id_key_deleted_at" from table: "plan_phases"
DROP INDEX "planphase_plan_id_key_deleted_at";
-- create index "planphase_plan_id_index_deleted_at" to table: "plan_phases"
CREATE UNIQUE INDEX "planphase_plan_id_index_deleted_at" ON "plan_phases" ("plan_id", "index", "deleted_at") WHERE (deleted_at IS NULL);
-- create index "planphase_plan_id_key_deleted_at" to table: "plan_phases"
CREATE UNIQUE INDEX "planphase_plan_id_key_deleted_at" ON "plan_phases" ("plan_id", "key", "deleted_at") WHERE (deleted_at IS NULL);
-- drop index "planratecard_phase_id_feature_key_deleted_at" from table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_feature_key_deleted_at";
-- drop index "planratecard_phase_id_key_deleted_at" from table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_key_deleted_at";
-- create index "planratecard_phase_id_feature_key_deleted_at" to table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_phase_id_feature_key_deleted_at" ON "plan_rate_cards" ("phase_id", "feature_key", "deleted_at") WHERE (deleted_at IS NULL);
-- create index "planratecard_phase_id_key_deleted_at" to table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_phase_id_key_deleted_at" ON "plan_rate_cards" ("phase_id", "key", "deleted_at") WHERE (deleted_at IS NULL);
-- drop index "plan_namespace_key_version_deleted_at" from table: "plans"
DROP INDEX "plan_namespace_key_version_deleted_at";
-- create index "plan_namespace_key_version_deleted_at" to table: "plans"
CREATE UNIQUE INDEX "plan_namespace_key_version_deleted_at" ON "plans" ("namespace", "key", "version", "deleted_at") WHERE (deleted_at IS NULL);
