-- reverse: create index "usagereset_namespace_entitlement_id_reset_time" to table: "usage_resets"
DROP INDEX "usagereset_namespace_entitlement_id_reset_time";
-- reverse: create index "usagereset_namespace_entitlement_id" to table: "usage_resets"
DROP INDEX "usagereset_namespace_entitlement_id";
-- reverse: create index "usagereset_id" to table: "usage_resets"
DROP INDEX "usagereset_id";
-- reverse: create "usage_resets" table
DROP TABLE "usage_resets";
-- reverse: create index "grant_namespace_owner_id" to table: "grants"
DROP INDEX "grant_namespace_owner_id";
-- reverse: create index "grant_id" to table: "grants"
DROP INDEX "grant_id";
-- reverse: create index "grant_effective_at_expires_at" to table: "grants"
DROP INDEX "grant_effective_at_expires_at";
-- reverse: create "grants" table
DROP TABLE "grants";
-- reverse: create index "balancesnapshot_namespace_balance_at" to table: "balance_snapshots"
DROP INDEX "balancesnapshot_namespace_balance_at";
-- reverse: create index "balancesnapshot_namespace_balance" to table: "balance_snapshots"
DROP INDEX "balancesnapshot_namespace_balance";
-- reverse: create index "balancesnapshot_namespace_at" to table: "balance_snapshots"
DROP INDEX "balancesnapshot_namespace_at";
-- reverse: create "balance_snapshots" table
DROP TABLE "balance_snapshots";
-- reverse: create index "entitlement_namespace_subject_key" to table: "entitlements"
DROP INDEX "entitlement_namespace_subject_key";
-- reverse: create index "entitlement_namespace_id_subject_key" to table: "entitlements"
DROP INDEX "entitlement_namespace_id_subject_key";
-- reverse: create index "entitlement_namespace_id" to table: "entitlements"
DROP INDEX "entitlement_namespace_id";
-- reverse: create index "entitlement_namespace_feature_id_id" to table: "entitlements"
DROP INDEX "entitlement_namespace_feature_id_id";
-- reverse: create index "entitlement_namespace_current_usage_period_end" to table: "entitlements"
DROP INDEX "entitlement_namespace_current_usage_period_end";
-- reverse: create index "entitlement_id" to table: "entitlements"
DROP INDEX "entitlement_id";
-- reverse: create "entitlements" table
DROP TABLE "entitlements";
-- reverse: create index "feature_namespace_id" to table: "features"
DROP INDEX "feature_namespace_id";
-- reverse: create index "feature_id" to table: "features"
DROP INDEX "feature_id";
-- reverse: create "features" table
DROP TABLE "features";
