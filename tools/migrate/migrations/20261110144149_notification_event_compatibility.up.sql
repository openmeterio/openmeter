-- modify "notification_event_delivery_status" table
ALTER TABLE "notification_event_delivery_status" ADD COLUMN "dispatch_generation" integer NOT NULL DEFAULT 1, ADD COLUMN "dispatch_queued_at" timestamptz NULL, ADD COLUMN "provider_data" jsonb NULL;
-- modify "notification_events" table
ALTER TABLE "notification_events" ALTER COLUMN "rule_id" DROP NOT NULL, ADD COLUMN "rule_snapshot" jsonb NULL, ADD COLUMN "provider_data" jsonb NULL;
