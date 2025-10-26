-- modify "notification_channels" table
ALTER TABLE "notification_channels" ADD COLUMN "annotations" jsonb NULL, ADD COLUMN "metadata" jsonb NULL;
-- create index "notificationchannel_annotations" to table: "notification_channels"
CREATE INDEX "notificationchannel_annotations" ON "notification_channels" USING gin ("annotations");
-- modify "notification_event_delivery_status" table
ALTER TABLE "notification_event_delivery_status" ADD COLUMN "annotations" jsonb NULL;
-- create index "notificationeventdeliverystatus_annotations" to table: "notification_event_delivery_status"
CREATE INDEX "notificationeventdeliverystatus_annotations" ON "notification_event_delivery_status" USING gin ("annotations");
-- modify "notification_rules" table
ALTER TABLE "notification_rules" ADD COLUMN "annotations" jsonb NULL, ADD COLUMN "metadata" jsonb NULL;
-- create index "notificationrule_annotations" to table: "notification_rules"
CREATE INDEX "notificationrule_annotations" ON "notification_rules" USING gin ("annotations");
