-- Run tools/migrate/notification_delivery_duplicate_preflight.sql and resolve any duplicates before upgrading.
-- atlas:nolint MF101
CREATE UNIQUE INDEX CONCURRENTLY "notification_delivery_destination_unique" ON "notification_event_delivery_status" ("namespace", "event_id", "channel_id");
