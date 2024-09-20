-- reverse: create index "usagereset_id" to table: "usage_resets"
DROP INDEX "usagereset_id";
-- reverse: drop index "usagereset_id" from table: "usage_resets"
CREATE INDEX "usagereset_id" ON "usage_resets" ("id");
-- reverse: create index "notificationrule_id" to table: "notification_rules"
DROP INDEX "notificationrule_id";
-- reverse: drop index "notificationrule_id" from table: "notification_rules"
CREATE INDEX "notificationrule_id" ON "notification_rules" ("id");
-- reverse: create index "notificationevent_id" to table: "notification_events"
DROP INDEX "notificationevent_id";
-- reverse: drop index "notificationevent_id" from table: "notification_events"
CREATE INDEX "notificationevent_id" ON "notification_events" ("id");
-- reverse: create index "notificationeventdeliverystatus_id" to table: "notification_event_delivery_status"
DROP INDEX "notificationeventdeliverystatus_id";
-- reverse: drop index "notificationeventdeliverystatus_id" from table: "notification_event_delivery_status"
CREATE INDEX "notificationeventdeliverystatus_id" ON "notification_event_delivery_status" ("id");
-- reverse: create index "notificationchannel_id" to table: "notification_channels"
DROP INDEX "notificationchannel_id";
-- reverse: drop index "notificationchannel_id" from table: "notification_channels"
CREATE INDEX "notificationchannel_id" ON "notification_channels" ("id");
-- reverse: create index "grant_id" to table: "grants"
DROP INDEX "grant_id";
-- reverse: drop index "grant_id" from table: "grants"
CREATE INDEX "grant_id" ON "grants" ("id");
-- reverse: create index "feature_id" to table: "features"
DROP INDEX "feature_id";
-- reverse: drop index "feature_id" from table: "features"
CREATE INDEX "feature_id" ON "features" ("id");
-- reverse: create index "entitlement_id" to table: "entitlements"
DROP INDEX "entitlement_id";
-- reverse: drop index "entitlement_id" from table: "entitlements"
CREATE INDEX "entitlement_id" ON "entitlements" ("id");
-- reverse: create index "customer_namespace_key_deleted_at" to table: "customers"
DROP INDEX "customer_namespace_key_deleted_at";
-- reverse: create index "customer_namespace_id" to table: "customers"
DROP INDEX "customer_namespace_id";
-- reverse: create index "customer_id" to table: "customers"
DROP INDEX "customer_id";
-- reverse: drop index "customer_namespace_key" from table: "customers"
CREATE UNIQUE INDEX "customer_namespace_key" ON "customers" ("namespace", "key");
-- reverse: drop index "customer_namespace_id" from table: "customers"
CREATE INDEX "customer_namespace_id" ON "customers" ("namespace", "id");
-- reverse: drop index "customer_id" from table: "customers"
CREATE INDEX "customer_id" ON "customers" ("id");
-- reverse: create index "billingworkflowconfig_id" to table: "billing_workflow_configs"
DROP INDEX "billingworkflowconfig_id";
-- reverse: drop index "billingworkflowconfig_id" from table: "billing_workflow_configs"
CREATE INDEX "billingworkflowconfig_id" ON "billing_workflow_configs" ("id");
-- reverse: create index "billingprofile_id" to table: "billing_profiles"
DROP INDEX "billingprofile_id";
-- reverse: drop index "billingprofile_id" from table: "billing_profiles"
CREATE INDEX "billingprofile_id" ON "billing_profiles" ("id");
-- reverse: create index "billinginvoice_id" to table: "billing_invoices"
DROP INDEX "billinginvoice_id";
-- reverse: drop index "billinginvoice_id" from table: "billing_invoices"
CREATE INDEX "billinginvoice_id" ON "billing_invoices" ("id");
-- reverse: create index "billinginvoiceitem_id" to table: "billing_invoice_items"
DROP INDEX "billinginvoiceitem_id";
-- reverse: drop index "billinginvoiceitem_id" from table: "billing_invoice_items"
CREATE INDEX "billinginvoiceitem_id" ON "billing_invoice_items" ("id");
