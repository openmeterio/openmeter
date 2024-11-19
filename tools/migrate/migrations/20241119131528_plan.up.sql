-- drop index "planratecard_phase_id_feature_key" from table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_feature_key";
-- drop index "planratecard_phase_id_key" from table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_key";
-- create index "planratecard_phase_id_feature_key_deleted_at" to table: "plan_rate_cards"
-- atlas:nolint
CREATE UNIQUE INDEX "planratecard_phase_id_feature_key_deleted_at" ON "plan_rate_cards" ("phase_id", "feature_key", "deleted_at");
-- create index "planratecard_phase_id_key_deleted_at" to table: "plan_rate_cards"
-- atlas:nolint
CREATE UNIQUE INDEX "planratecard_phase_id_key_deleted_at" ON "plan_rate_cards" ("phase_id", "key", "deleted_at");
