BEGIN;

-- Capture the schema-level-1 rows and their approximate predecessors before
-- updating them. Voided history is intentionally included: ordering all runs
-- by creation time per charge is the accepted approximation for this backfill.
CREATE TEMP TABLE "om_usage_based_run_prior_lineage_backfill" (
  "id" character(26) PRIMARY KEY,
  "prior_run_id" character(26) NULL
) ON COMMIT DROP;

INSERT INTO "om_usage_based_run_prior_lineage_backfill" ("id", "prior_run_id")
SELECT "id", "prior_run_id"
FROM (
  SELECT
    "id",
    "schema_level",
    LAG("id") OVER (
      PARTITION BY "namespace", "charge_id"
      ORDER BY "created_at", "id"
    ) AS "prior_run_id"
  FROM "charge_usage_based_runs"
) AS "ordered_runs"
WHERE "schema_level" = 1
ORDER BY "id";

-- schema_level is the commit marker and is promoted only together with the
-- lineage value captured above. Existing schema-level-2 rows are untouched.
UPDATE "charge_usage_based_runs" AS "run"
SET
  "prior_run_id" = "backfill"."prior_run_id",
  "schema_level" = 2
FROM "om_usage_based_run_prior_lineage_backfill" AS "backfill"
WHERE "run"."id" = "backfill"."id"
  AND "run"."schema_level" = 1;

DO $migration$
DECLARE
  invalid_ids TEXT;
BEGIN
  SELECT string_agg("run"."id"::text, ', ' ORDER BY "run"."id")
  INTO invalid_ids
  FROM "om_usage_based_run_prior_lineage_backfill" AS "backfill"
  JOIN "charge_usage_based_runs" AS "run" ON "run"."id" = "backfill"."id"
  WHERE "run"."schema_level" IS DISTINCT FROM 2
    OR "run"."prior_run_id" IS DISTINCT FROM "backfill"."prior_run_id";

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'usage-based realization run lineage backfill is incomplete [ids=%]', invalid_ids;
  END IF;

  SELECT string_agg("id"::text, ', ' ORDER BY "id")
  INTO invalid_ids
  FROM "charge_usage_based_runs"
  WHERE "schema_level" = 1;

  IF invalid_ids IS NOT NULL THEN
    RAISE EXCEPTION 'usage-based realization runs remain at schema level 1 [ids=%]', invalid_ids;
  END IF;
END
$migration$;

COMMIT;
