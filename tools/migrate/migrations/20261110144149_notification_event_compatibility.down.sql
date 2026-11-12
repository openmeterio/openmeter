-- reverse: modify "notification_events" table
ALTER TABLE "notification_events" DROP COLUMN "provider_data", DROP COLUMN "rule_snapshot", ALTER COLUMN "rule_id" SET NOT NULL;
-- reverse: modify "notification_event_delivery_status" table
ALTER TABLE "notification_event_delivery_status" DROP COLUMN "provider_data", DROP COLUMN "dispatch_queued_at", DROP COLUMN "dispatch_generation";
