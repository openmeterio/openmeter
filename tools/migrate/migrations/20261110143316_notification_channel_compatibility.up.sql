-- modify "notification_channels" table
ALTER TABLE "notification_channels" ADD COLUMN "event_type_filter" jsonb NULL, ADD COLUMN "provider_data" jsonb NULL, ADD COLUMN "generation" integer NOT NULL DEFAULT 1, ADD COLUMN "reconcile_status" text NOT NULL DEFAULT 'unknown', ADD COLUMN "reconcile_next_at" timestamptz NULL;
