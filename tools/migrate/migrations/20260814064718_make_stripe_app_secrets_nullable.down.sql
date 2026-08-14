-- reverse: modify "app_stripes" table
ALTER TABLE "app_stripes" ALTER COLUMN "webhook_secret" SET NOT NULL, ALTER COLUMN "api_key" SET NOT NULL, DROP CONSTRAINT "app_stripe_secret_lifecycle";
