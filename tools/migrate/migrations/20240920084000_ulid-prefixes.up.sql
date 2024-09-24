-- modify "balance_snapshots" table
ALTER TABLE "balance_snapshots" ALTER COLUMN "owner_id" TYPE character varying(34);
-- modify "billing_invoice_items" table
ALTER TABLE "billing_invoice_items" ALTER COLUMN "id" TYPE character varying(34), ALTER COLUMN "invoice_id" TYPE character varying(34);
-- modify "billing_invoices" table
ALTER TABLE "billing_invoices" ALTER COLUMN "id" TYPE character varying(34), ALTER COLUMN "billing_profile_id" TYPE character varying(34), ALTER COLUMN "workflow_config_id" TYPE character varying(34);
-- modify "billing_profiles" table
ALTER TABLE "billing_profiles" ALTER COLUMN "id" TYPE character varying(34), ALTER COLUMN "workflow_config_id" TYPE character varying(34);
-- modify "billing_workflow_configs" table
ALTER TABLE "billing_workflow_configs" ALTER COLUMN "id" TYPE character varying(34);
-- modify "customer_subjects" table
ALTER TABLE "customer_subjects" ALTER COLUMN "customer_id" TYPE character varying(34);
-- modify "customers" table
ALTER TABLE "customers" ALTER COLUMN "id" TYPE character varying(34);
-- modify "entitlements" table
ALTER TABLE "entitlements" ALTER COLUMN "id" TYPE character varying(34), ALTER COLUMN "feature_id" TYPE character varying(34);
-- modify "features" table
ALTER TABLE "features" ALTER COLUMN "id" TYPE character varying(34);
-- modify "grants" table
ALTER TABLE "grants" ALTER COLUMN "id" TYPE character varying(34), ALTER COLUMN "owner_id" TYPE character varying(34);
-- modify "notification_channel_rules" table
ALTER TABLE "notification_channel_rules" ALTER COLUMN "notification_channel_id" TYPE character varying(34), ALTER COLUMN "notification_rule_id" TYPE character varying(34);
-- modify "notification_channels" table
ALTER TABLE "notification_channels" ALTER COLUMN "id" TYPE character varying(34);
-- modify "notification_event_delivery_status" table
ALTER TABLE "notification_event_delivery_status" ALTER COLUMN "id" TYPE character varying(34);
-- modify "notification_event_delivery_status_events" table
ALTER TABLE "notification_event_delivery_status_events" ALTER COLUMN "notification_event_delivery_status_id" TYPE character varying(34), ALTER COLUMN "notification_event_id" TYPE character varying(34);
-- modify "notification_events" table
ALTER TABLE "notification_events" ALTER COLUMN "id" TYPE character varying(34), ALTER COLUMN "rule_id" TYPE character varying(34);
-- modify "notification_rules" table
ALTER TABLE "notification_rules" ALTER COLUMN "id" TYPE character varying(34);
-- modify "usage_resets" table
ALTER TABLE "usage_resets" ALTER COLUMN "id" TYPE character varying(34), ALTER COLUMN "entitlement_id" TYPE character varying(34);
