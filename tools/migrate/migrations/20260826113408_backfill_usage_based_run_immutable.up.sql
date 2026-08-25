-- Historical usage-based realization runs predate the immutable flag. Mark runs
-- immutable only when invoiced usage exists and the linked invoice has clear
-- evidence that issuance completed. In-progress and failed issuing states are
-- intentionally excluded unless the invoice was sent to the customer.
BEGIN;

WITH marked_usages AS (
  UPDATE charge_usage_based_run_invoiced_usages AS usage
  SET
    annotations =
      CASE
        WHEN usage.annotations IS NULL OR usage.annotations = 'null'::jsonb
          THEN '{}'::jsonb
        ELSE usage.annotations
      END ||
      jsonb_build_object(
        'dbmigration:backfill_usage_based_run_immutable',
        to_char(
          CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
          'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
        )
      ),
    updated_at = CURRENT_TIMESTAMP
  FROM charge_usage_based_runs AS run
  JOIN billing_invoices AS invoice
    ON invoice.id = run.invoice_id
   AND invoice.namespace = run.namespace
  WHERE usage.run_id = run.id
    AND usage.namespace = run.namespace
    AND NOT run.immutable
    AND NOT COALESCE(
      usage.annotations ? 'dbmigration:backfill_usage_based_run_immutable',
      false
    )
    AND (
      invoice.sent_to_customer_at IS NOT NULL
      OR invoice.status = 'issued'
      OR invoice.status LIKE 'payment_processing.%'
      OR invoice.status IN ('overdue', 'paid', 'uncollectible', 'voided')
    )
  RETURNING usage.run_id, usage.namespace
)
UPDATE charge_usage_based_runs AS run
SET
  immutable = true,
  updated_at = CURRENT_TIMESTAMP
FROM marked_usages AS usage
WHERE run.id = usage.run_id
  AND run.namespace = usage.namespace;

COMMIT;
