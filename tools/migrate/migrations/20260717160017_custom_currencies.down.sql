-- reverse: modify "custom_currencies" table
ALTER TABLE "custom_currencies" DROP COLUMN "thousands_separator", DROP COLUMN "decimal_mark", DROP COLUMN "precision", ALTER COLUMN "symbol" SET NOT NULL;
