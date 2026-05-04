-- modify "charge_usage_based_runs" table
ALTER TABLE "charge_usage_based_runs" ADD COLUMN "detailed_lines_present" boolean NOT NULL DEFAULT false;
UPDATE "charge_usage_based_runs" AS "runs"
SET "detailed_lines_present" = true
WHERE EXISTS (
  SELECT 1
  FROM "charge_usage_based_run_detailed_line" AS "lines"
  WHERE "lines"."namespace" = "runs"."namespace"
    AND "lines"."charge_id" = "runs"."charge_id"
    AND "lines"."run_id" = "runs"."id"
    AND "lines"."deleted_at" IS NULL
);
UPDATE "charge_usage_based_runs" AS "runs"
SET "detailed_lines_present" = true
FROM "charge_usage_based" AS "charges"
WHERE "charges"."namespace" = "runs"."namespace"
  AND "charges"."id" = "runs"."charge_id"
  AND "charges"."status_detailed" = 'final';
ALTER TABLE "charge_usage_based_runs" ALTER COLUMN "detailed_lines_present" DROP DEFAULT;
