BEGIN;

UPDATE charge_usage_based_runs AS run
SET
  immutable = false,
  updated_at = CURRENT_TIMESTAMP
FROM charge_usage_based_run_invoiced_usages AS usage
WHERE usage.run_id = run.id
  AND usage.namespace = run.namespace
  AND COALESCE(
    usage.annotations ? 'dbmigration:backfill_usage_based_run_immutable',
    false
  );

UPDATE charge_usage_based_run_invoiced_usages
SET
  annotations = CASE
    WHEN annotations - 'dbmigration:backfill_usage_based_run_immutable' = '{}'::jsonb
      THEN NULL
    ELSE annotations - 'dbmigration:backfill_usage_based_run_immutable'
  END,
  updated_at = CURRENT_TIMESTAMP
WHERE COALESCE(
  annotations ? 'dbmigration:backfill_usage_based_run_immutable',
  false
);

COMMIT;
