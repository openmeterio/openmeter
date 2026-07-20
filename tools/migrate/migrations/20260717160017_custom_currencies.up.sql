-- modify "custom_currencies" table
ALTER TABLE "custom_currencies" ALTER COLUMN "symbol" DROP NOT NULL, ADD COLUMN "precision" bigint NOT NULL DEFAULT 2, ADD COLUMN "decimal_mark" character varying NOT NULL DEFAULT '.', ADD COLUMN "thousands_separator" character varying NOT NULL DEFAULT ',';
