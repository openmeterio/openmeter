-- modify "app_stripes" table
ALTER TABLE "app_stripes" ADD COLUMN "webhook_schema_version" bigint NOT NULL DEFAULT 1;
