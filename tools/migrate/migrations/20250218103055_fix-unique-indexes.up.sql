-- drop index "appstripe_namespace_stripe_account_id_stripe_livemode" from table: "app_stripes"
DROP INDEX "appstripe_namespace_stripe_account_id_stripe_livemode";
-- create index "appstripe_namespace_stripe_account_id_stripe_livemode" to table: "app_stripes"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "appstripe_namespace_stripe_account_id_stripe_livemode" ON "app_stripes" ("namespace", "stripe_account_id", "stripe_livemode") WHERE (deleted_at IS NULL);
-- drop index "billingprofile_namespace_default_deleted_at" from table: "billing_profiles"
DROP INDEX "billingprofile_namespace_default_deleted_at";
-- create index "billingprofile_namespace_default" to table: "billing_profiles"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billingprofile_namespace_default" ON "billing_profiles" ("namespace", "default") WHERE (deleted_at IS NULL);
-- create index "feature_namespace_key" to table: "features"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "feature_namespace_key" ON "features" ("namespace", "key") WHERE (archived_at IS NULL);
-- drop index "planphase_plan_id_index_deleted_at" from table: "plan_phases"
DROP INDEX "planphase_plan_id_index_deleted_at";
-- drop index "planphase_plan_id_key_deleted_at" from table: "plan_phases"
DROP INDEX "planphase_plan_id_key_deleted_at";
-- create index "planphase_plan_id_index" to table: "plan_phases"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "planphase_plan_id_index" ON "plan_phases" ("plan_id", "index") WHERE (deleted_at IS NULL);
-- create index "planphase_plan_id_key" to table: "plan_phases"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "planphase_plan_id_key" ON "plan_phases" ("plan_id", "key") WHERE (deleted_at IS NULL);
-- drop index "planratecard_phase_id_feature_key_deleted_at" from table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_feature_key_deleted_at";
-- drop index "planratecard_phase_id_key_deleted_at" from table: "plan_rate_cards"
DROP INDEX "planratecard_phase_id_key_deleted_at";
-- create index "planratecard_phase_id_feature_key" to table: "plan_rate_cards"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "planratecard_phase_id_feature_key" ON "plan_rate_cards" ("phase_id", "feature_key") WHERE (deleted_at IS NULL);
-- create index "planratecard_phase_id_key" to table: "plan_rate_cards"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "planratecard_phase_id_key" ON "plan_rate_cards" ("phase_id", "key") WHERE (deleted_at IS NULL);
-- drop index "plan_namespace_key_version_deleted_at" from table: "plans"
DROP INDEX "plan_namespace_key_version_deleted_at";
-- create index "plan_namespace_key_version" to table: "plans"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "plan_namespace_key_version" ON "plans" ("namespace", "key", "version") WHERE (deleted_at IS NULL);
