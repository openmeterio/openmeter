-- reverse: create index "notificationeventdeliverystatus_namespace_state_next_attempt_at" to table: "notification_event_delivery_status"
DROP INDEX "notificationeventdeliverystatus_namespace_state_next_attempt_at";
-- reverse: modify "notification_event_delivery_status" table
ALTER TABLE "notification_event_delivery_status" DROP COLUMN "attempts", DROP COLUMN "next_attempt_at";
