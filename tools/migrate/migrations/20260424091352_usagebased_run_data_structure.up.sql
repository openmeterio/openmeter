-- modify "charge_usage_based_runs" table
-- atlas:nolint BC102
ALTER TABLE "charge_usage_based_runs" RENAME COLUMN "asof" TO "stored_at_lt";
-- atlas:nolint BC102
ALTER TABLE "charge_usage_based_runs" RENAME COLUMN "meter_value" TO "metered_quantity";
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "service_period_to" timestamptz;
-- Existing runtime snapshots used collection_end as the actual metering cutoff; asof was only
-- the run creation timestamp. Preserve the query cutoff under the new stored_at_lt name.
UPDATE "charge_usage_based_runs" SET "stored_at_lt" = "collection_end";
-- Final runs can backfill their service period from the immutable parent charge intent. Partial
-- runs should not exist in persisted production data yet, but keep a safe fallback to the old
-- collection_end value for any non-final rows.
UPDATE "charge_usage_based_runs" AS "r"
SET "service_period_to" = CASE
    WHEN "r"."type" = 'final_realization' THEN "c"."service_period_to"
    ELSE "r"."collection_end"
END
FROM "charge_usage_based" AS "c"
WHERE "r"."namespace" = "c"."namespace"
  AND "r"."charge_id" = "c"."id"
  AND "r"."service_period_to" IS NULL;
-- atlas:nolint MF104
ALTER TABLE "charge_usage_based_runs" ALTER COLUMN "service_period_to" SET NOT NULL;
-- atlas:nolint DS103
ALTER TABLE "charge_usage_based_runs" DROP COLUMN "collection_end";
