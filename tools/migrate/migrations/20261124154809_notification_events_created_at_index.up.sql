-- create index "notificationevent_namespace_created_at_id" to table: "notification_events"
CREATE INDEX CONCURRENTLY IF NOT EXISTS "notificationevent_namespace_created_at_id" ON "notification_events" ("namespace", "created_at", "id");
