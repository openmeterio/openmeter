-- reverse: create index "planratecard_phase_id_key_deleted_at" to table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_key_deleted_at";
-- reverse: create index "planratecard_phase_id_feature_key_deleted_at" to table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_feature_key_deleted_at";
-- reverse: drop index "planratecard_phase_id_key" from table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_phase_id_key" ON "plan_rate_cards" ("phase_id", "key");
-- reverse: drop index "planratecard_phase_id_feature_key" from table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_phase_id_feature_key" ON "plan_rate_cards" ("phase_id", "feature_key");
