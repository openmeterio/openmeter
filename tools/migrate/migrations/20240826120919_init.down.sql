-- reverse: create index "usagereset_namespace_entitlement_id_reset_time" to table: "usage_resets"
DROP INDEX "usagereset_namespace_entitlement_id_reset_time";
-- reverse: create index "usagereset_namespace_entitlement_id" to table: "usage_resets"
DROP INDEX "usagereset_namespace_entitlement_id";
-- reverse: create index "usagereset_id" to table: "usage_resets"
DROP INDEX "usagereset_id";
-- reverse: create "usage_resets" table
DROP TABLE "usage_resets";
-- reverse: create "notification_event_delivery_status_events" table
DROP TABLE "notification_event_delivery_status_events";
-- reverse: create index "notificationevent_namespace_type" to table: "notification_events"
DROP INDEX "notificationevent_namespace_type";
-- reverse: create index "notificationevent_namespace_id" to table: "notification_events"
DROP INDEX "notificationevent_namespace_id";
-- reverse: create index "notificationevent_id" to table: "notification_events"
DROP INDEX "notificationevent_id";
-- reverse: create index "notificationevent_annotations" to table: "notification_events"
DROP INDEX "notificationevent_annotations";
-- reverse: create "notification_events" table
DROP TABLE "notification_events";
-- reverse: create index "notificationeventdeliverystatus_namespace_state" to table: "notification_event_delivery_status"
DROP INDEX "notificationeventdeliverystatus_namespace_state";
-- reverse: create index "notificationeventdeliverystatus_namespace_id" to table: "notification_event_delivery_status"
DROP INDEX "notificationeventdeliverystatus_namespace_id";
-- reverse: create index "notificationeventdeliverystatus_namespace_event_id_channel_id" to table: "notification_event_delivery_status"
DROP INDEX "notificationeventdeliverystatus_namespace_event_id_channel_id";
-- reverse: create index "notificationeventdeliverystatus_id" to table: "notification_event_delivery_status"
DROP INDEX "notificationeventdeliverystatus_id";
-- reverse: create "notification_event_delivery_status" table
DROP TABLE "notification_event_delivery_status";
-- reverse: create "notification_channel_rules" table
DROP TABLE "notification_channel_rules";
-- reverse: create index "notificationrule_namespace_type" to table: "notification_rules"
DROP INDEX "notificationrule_namespace_type";
-- reverse: create index "notificationrule_namespace_id" to table: "notification_rules"
DROP INDEX "notificationrule_namespace_id";
-- reverse: create index "notificationrule_id" to table: "notification_rules"
DROP INDEX "notificationrule_id";
-- reverse: create "notification_rules" table
DROP TABLE "notification_rules";
-- reverse: create index "notificationchannel_namespace_type" to table: "notification_channels"
DROP INDEX "notificationchannel_namespace_type";
-- reverse: create index "notificationchannel_namespace_id" to table: "notification_channels"
DROP INDEX "notificationchannel_namespace_id";
-- reverse: create index "notificationchannel_id" to table: "notification_channels"
DROP INDEX "notificationchannel_id";
-- reverse: create "notification_channels" table
DROP TABLE "notification_channels";
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
