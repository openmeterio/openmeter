-- reverse: create index "notificationrule_annotations" to table: "notification_rules"
DROP INDEX "notificationrule_annotations";
-- reverse: modify "notification_rules" table
ALTER TABLE "notification_rules" DROP COLUMN "metadata", DROP COLUMN "annotations";
-- reverse: create index "notificationeventdeliverystatus_annotations" to table: "notification_event_delivery_status"
DROP INDEX "notificationeventdeliverystatus_annotations";
-- reverse: modify "notification_event_delivery_status" table
ALTER TABLE "notification_event_delivery_status" DROP COLUMN "annotations";
-- reverse: create index "notificationchannel_annotations" to table: "notification_channels"
DROP INDEX "notificationchannel_annotations";
-- reverse: modify "notification_channels" table
ALTER TABLE "notification_channels" DROP COLUMN "metadata", DROP COLUMN "annotations";
