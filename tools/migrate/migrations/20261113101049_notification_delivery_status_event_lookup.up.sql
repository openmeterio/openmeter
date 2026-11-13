-- create index "notification_status_events_event_status_idx" to table: "notification_event_delivery_status_events"
CREATE INDEX CONCURRENTLY "notification_status_events_event_status_idx" ON "notification_event_delivery_status_events" ("notification_event_id", "notification_event_delivery_status_id");
