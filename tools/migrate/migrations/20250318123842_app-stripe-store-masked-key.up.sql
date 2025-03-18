-- modify "app_stripes" table
ALTER TABLE "app_stripes" ADD COLUMN "masked_api_key" character varying NOT NULL DEFAULT 'sk_***';
ALTER TABLE "app_stripes" ALTER COLUMN "masked_api_key" DROP DEFAULT;
