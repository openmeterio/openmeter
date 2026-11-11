-- reverse: modify "notification_channels" table
ALTER TABLE "notification_channels" DROP COLUMN "reconcile_next_at", DROP COLUMN "reconcile_status", DROP COLUMN "generation", DROP COLUMN "provider_data", DROP COLUMN "event_type_filter";
