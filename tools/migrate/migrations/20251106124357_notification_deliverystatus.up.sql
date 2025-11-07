-- modify "notification_event_delivery_status" table
ALTER TABLE "notification_event_delivery_status" ADD COLUMN "next_attempt_at" timestamptz NULL, ADD COLUMN "attempts" jsonb NULL;
-- create index "notificationeventdeliverystatus_namespace_state_next_attempt_at" to table: "notification_event_delivery_status"
CREATE INDEX "notificationeventdeliverystatus_namespace_state_next_attempt_at" ON "notification_event_delivery_status" ("namespace", "state", "next_attempt_at");
