-- reverse: create index "customersubjects_customer_id_subject_key" to table: "customer_subjects"
DROP INDEX "customersubjects_customer_id_subject_key";
-- reverse: create "customer_subjects" table
DROP TABLE "customer_subjects";
-- reverse: create index "customer_namespace_key" to table: "customers"
DROP INDEX "customer_namespace_key";
-- reverse: create index "customer_namespace_id" to table: "customers"
DROP INDEX "customer_namespace_id";
-- reverse: create index "customer_namespace" to table: "customers"
DROP INDEX "customer_namespace";
-- reverse: create index "customer_id" to table: "customers"
DROP INDEX "customer_id";
-- reverse: create "customers" table
DROP TABLE "customers";
-- reverse: create index "usagereset_namespace" to table: "usage_resets"
DROP INDEX "usagereset_namespace";
-- reverse: create index "notificationeventdeliverystatus_namespace" to table: "notification_event_delivery_status"
DROP INDEX "notificationeventdeliverystatus_namespace";
-- reverse: create index "notificationevent_namespace" to table: "notification_events"
DROP INDEX "notificationevent_namespace";
-- reverse: create index "notificationrule_namespace" to table: "notification_rules"
DROP INDEX "notificationrule_namespace";
-- reverse: create index "notificationchannel_namespace" to table: "notification_channels"
DROP INDEX "notificationchannel_namespace";
-- reverse: create index "billinginvoiceitem_namespace" to table: "billing_invoice_items"
DROP INDEX "billinginvoiceitem_namespace";
-- reverse: create index "entitlement_namespace" to table: "entitlements"
DROP INDEX "entitlement_namespace";
-- reverse: create index "billingworkflowconfig_namespace" to table: "billing_workflow_configs"
DROP INDEX "billingworkflowconfig_namespace";
-- reverse: create index "billingprofile_namespace" to table: "billing_profiles"
DROP INDEX "billingprofile_namespace";
-- reverse: create index "billinginvoice_namespace" to table: "billing_invoices"
DROP INDEX "billinginvoice_namespace";
-- reverse: create index "balancesnapshot_namespace" to table: "balance_snapshots"
DROP INDEX "balancesnapshot_namespace";
-- reverse: create index "grant_namespace" to table: "grants"
DROP INDEX "grant_namespace";
