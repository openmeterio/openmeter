-- reverse: create index "notificationeventdeliverystatus_namespace_event_id_channel_id" to table: "notification_event_delivery_status"
DROP INDEX "notificationeventdeliverystatus_namespace_event_id_channel_id";
-- reverse: drop index "notificationeventdeliverystatus_namespace_event_id_channel_id" from table: "notification_event_delivery_status"
CREATE INDEX "notificationeventdeliverystatus_namespace_event_id_channel_id" ON "notification_event_delivery_status" ("namespace", "event_id", "channel_id");
