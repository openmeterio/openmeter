-- reverse: modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "collection_end" timestamptz;
UPDATE "charge_usage_based_runs" SET "collection_end" = "stored_at_lt";
ALTER TABLE "charge_usage_based_runs" ALTER COLUMN "collection_end" SET NOT NULL;
ALTER TABLE "charge_usage_based_runs" DROP COLUMN "service_period_to";
ALTER TABLE "charge_usage_based_runs" RENAME COLUMN "metered_quantity" TO "meter_value";
ALTER TABLE "charge_usage_based_runs" RENAME COLUMN "stored_at_lt" TO "asof";
