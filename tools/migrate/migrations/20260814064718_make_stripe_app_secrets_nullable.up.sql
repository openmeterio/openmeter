-- modify "app_stripes" table
ALTER TABLE "app_stripes" ADD CONSTRAINT "app_stripe_secret_lifecycle" CHECK (((deleted_at IS NULL) AND (api_key IS NOT NULL) AND (webhook_secret IS NOT NULL)) OR ((deleted_at IS NOT NULL) AND (api_key IS NULL) AND (webhook_secret IS NULL))), ALTER COLUMN "api_key" DROP NOT NULL, ALTER COLUMN "webhook_secret" DROP NOT NULL;
