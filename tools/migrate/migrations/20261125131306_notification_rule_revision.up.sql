-- modify "notification_rules" table
ALTER TABLE "notification_rules" ADD COLUMN "revision" integer NOT NULL DEFAULT 1;
