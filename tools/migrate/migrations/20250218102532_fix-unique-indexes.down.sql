-- reverse: create index "plan_namespace_key_version" to table: "plans"
DROP INDEX "plan_namespace_key_version";
-- reverse: drop index "plan_namespace_key_version_deleted_at" from table: "plans"
CREATE UNIQUE INDEX "plan_namespace_key_version_deleted_at" ON "plans" ("namespace", "key", "version", "deleted_at");
-- reverse: create index "planratecard_phase_id_key" to table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_key";
-- reverse: create index "planratecard_phase_id_feature_key" to table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_feature_key";
-- reverse: drop index "planratecard_phase_id_key_deleted_at" from table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_phase_id_key_deleted_at" ON "plan_rate_cards" ("phase_id", "key", "deleted_at");
-- reverse: drop index "planratecard_phase_id_feature_key_deleted_at" from table: "plan_rate_cards"
CREATE UNIQUE INDEX "planratecard_phase_id_feature_key_deleted_at" ON "plan_rate_cards" ("phase_id", "feature_key", "deleted_at");
-- reverse: create index "planphase_plan_id_key" to table: "plan_phases"
DROP INDEX "planphase_plan_id_key";
-- reverse: create index "planphase_plan_id_index" to table: "plan_phases"
DROP INDEX "planphase_plan_id_index";
-- reverse: drop index "planphase_plan_id_key_deleted_at" from table: "plan_phases"
CREATE UNIQUE INDEX "planphase_plan_id_key_deleted_at" ON "plan_phases" ("plan_id", "key", "deleted_at");
-- reverse: drop index "planphase_plan_id_index_deleted_at" from table: "plan_phases"
CREATE UNIQUE INDEX "planphase_plan_id_index_deleted_at" ON "plan_phases" ("plan_id", "index", "deleted_at");
-- reverse: create index "feature_namespace_key" to table: "features"
DROP INDEX "feature_namespace_key";
-- reverse: create index "appstripe_namespace_stripe_account_id_stripe_livemode" to table: "app_stripes"
DROP INDEX "appstripe_namespace_stripe_account_id_stripe_livemode";
-- reverse: drop index "appstripe_namespace_stripe_account_id_stripe_livemode" from table: "app_stripes"
CREATE UNIQUE INDEX "appstripe_namespace_stripe_account_id_stripe_livemode" ON "app_stripes" ("namespace", "stripe_account_id", "stripe_livemode");
