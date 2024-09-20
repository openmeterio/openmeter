
-- drop index "billinginvoiceitem_id" from table: "billing_invoice_items"
DROP INDEX "billinginvoiceitem_id";
-- create index "billinginvoiceitem_id" to table: "billing_invoice_items"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billinginvoiceitem_id" ON "billing_invoice_items" ("id");
-- drop index "billinginvoice_id" from table: "billing_invoices"
DROP INDEX "billinginvoice_id";
-- create index "billinginvoice_id" to table: "billing_invoices"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billinginvoice_id" ON "billing_invoices" ("id");
-- drop index "billingprofile_id" from table: "billing_profiles"
DROP INDEX "billingprofile_id";
-- create index "billingprofile_id" to table: "billing_profiles"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billingprofile_id" ON "billing_profiles" ("id");
-- drop index "billingworkflowconfig_id" from table: "billing_workflow_configs"
DROP INDEX "billingworkflowconfig_id";
-- create index "billingworkflowconfig_id" to table: "billing_workflow_configs"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "billingworkflowconfig_id" ON "billing_workflow_configs" ("id");
-- drop index "customer_id" from table: "customers"
DROP INDEX "customer_id";
-- drop index "customer_namespace_id" from table: "customers"
DROP INDEX "customer_namespace_id";
-- drop index "customer_namespace_key" from table: "customers"
DROP INDEX "customer_namespace_key";
-- create index "customer_id" to table: "customers"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "customer_id" ON "customers" ("id");
-- create index "customer_namespace_id" to table: "customers"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "customer_namespace_id" ON "customers" ("namespace", "id");
-- create index "customer_namespace_key_deleted_at" to table: "customers"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "customer_namespace_key_deleted_at" ON "customers" ("namespace", "key", "deleted_at");
-- drop index "entitlement_id" from table: "entitlements"
DROP INDEX "entitlement_id";
-- create index "entitlement_id" to table: "entitlements"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "entitlement_id" ON "entitlements" ("id");
-- drop index "feature_id" from table: "features"
DROP INDEX "feature_id";
-- create index "feature_id" to table: "features"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "feature_id" ON "features" ("id");
-- drop index "grant_id" from table: "grants"
DROP INDEX "grant_id";
-- create index "grant_id" to table: "grants"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "grant_id" ON "grants" ("id");
-- drop index "notificationchannel_id" from table: "notification_channels"
DROP INDEX "notificationchannel_id";
-- create index "notificationchannel_id" to table: "notification_channels"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "notificationchannel_id" ON "notification_channels" ("id");
-- drop index "notificationeventdeliverystatus_id" from table: "notification_event_delivery_status"
DROP INDEX "notificationeventdeliverystatus_id";
-- create index "notificationeventdeliverystatus_id" to table: "notification_event_delivery_status"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "notificationeventdeliverystatus_id" ON "notification_event_delivery_status" ("id");
-- drop index "notificationevent_id" from table: "notification_events"
DROP INDEX "notificationevent_id";
-- create index "notificationevent_id" to table: "notification_events"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "notificationevent_id" ON "notification_events" ("id");
-- drop index "notificationrule_id" from table: "notification_rules"
DROP INDEX "notificationrule_id";
-- create index "notificationrule_id" to table: "notification_rules"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "notificationrule_id" ON "notification_rules" ("id");
-- drop index "usagereset_id" from table: "usage_resets"
DROP INDEX "usagereset_id";
-- create index "usagereset_id" to table: "usage_resets"
-- atlas:nolint MF101
CREATE UNIQUE INDEX "usagereset_id" ON "usage_resets" ("id");
