CREATE UNIQUE INDEX CONCURRENTLY "notification_delivery_destination_unique" ON "notification_event_delivery_status" ("namespace", "event_id", "channel_id");
