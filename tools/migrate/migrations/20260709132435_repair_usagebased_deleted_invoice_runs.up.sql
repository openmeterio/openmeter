-- Repair usage-based realization runs left live after API-deleting progressively billed standard invoices.
--
-- The original API delete path removed the invoice/line but did not dispatch the usage-based
-- line-engine cleanup for these rows. Restrict the repair to zero-credit runs so no ledger credit
-- correction is needed:
-- - partial_invoice runs are only soft-deleted;
-- - final_realization runs are soft-deleted and the charge is made effectively deleted through
--   an override intent, leaving the subscription-owned base intent intact for subscription sync;
-- - gathering lines for deleted charges are soft-deleted so subscription sync does not keep
--   referencing charges that the repair intentionally tombstones;
-- - touched gathering invoices are soft-deleted when those line deletions leave no live lines.
BEGIN;

CREATE TEMPORARY TABLE repair_usagebased_deleted_invoice_runs ON COMMIT DROP AS
SELECT
  l.namespace,
  l.charge_id,
  r.id AS run_id,
  r.type AS run_type
FROM billing_invoice_lines l
JOIN charge_usage_based_runs r
  ON r.namespace = l.namespace
 AND r.charge_id = l.charge_id
 AND r.line_id = l.id
JOIN billing_invoices i
  ON i.namespace = l.namespace
 AND i.id = l.invoice_id
WHERE l.charge_id IS NOT NULL
  AND l.status = 'valid'
  AND (
    l.deleted_at IS NOT NULL
    OR (i.deleted_at IS NOT NULL AND i.status <> 'gathering')
  )
  AND r.deleted_at IS NULL
  AND r.credits_total = 0
  AND r.type IN ('partial_invoice', 'final_realization');

UPDATE charge_usage_based_runs r
SET
  deleted_at = now(),
  updated_at = now()
FROM repair_usagebased_deleted_invoice_runs affected
WHERE r.namespace = affected.namespace
  AND r.id = affected.run_id;

UPDATE charge_usage_based c
SET
  current_realization_run_id = NULL,
  updated_at = now()
FROM repair_usagebased_deleted_invoice_runs affected
WHERE c.namespace = affected.namespace
  AND c.id = affected.charge_id
  AND c.current_realization_run_id = affected.run_id;

UPDATE billing_invoice_lines l
SET
  deleted_at = now(),
  updated_at = now()
FROM (
  SELECT DISTINCT namespace, charge_id
  FROM repair_usagebased_deleted_invoice_runs
  WHERE run_type = 'final_realization'
) affected,
billing_invoices i
WHERE l.namespace = affected.namespace
  AND l.charge_id = affected.charge_id
  AND l.deleted_at IS NULL
  AND l.status = 'valid'
  AND i.namespace = l.namespace
  AND i.id = l.invoice_id
  AND i.status = 'gathering'
  AND i.deleted_at IS NULL;

UPDATE billing_invoices i
SET
  deleted_at = now(),
  updated_at = now()
FROM (
  SELECT DISTINCT l.namespace, l.invoice_id
  FROM billing_invoice_lines l
  JOIN repair_usagebased_deleted_invoice_runs affected
    ON affected.namespace = l.namespace
   AND affected.charge_id = l.charge_id
   AND affected.run_type = 'final_realization'
) touched
WHERE i.namespace = touched.namespace
  AND i.id = touched.invoice_id
  AND i.status = 'gathering'
  AND i.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM billing_invoice_lines remaining
    WHERE remaining.namespace = i.namespace
      AND remaining.invoice_id = i.id
      AND remaining.deleted_at IS NULL
  );

INSERT INTO charge_usage_based_overrides (
  id,
  namespace,
  charge_id,
  name,
  description,
  metadata,
  tax_behavior,
  tax_code_id,
  intent_deleted_at,
  service_period_from,
  service_period_to,
  full_service_period_from,
  full_service_period_to,
  billing_period_from,
  billing_period_to,
  invoice_at,
  feature_key,
  price,
  discounts,
  unit_config
)
SELECT
  c.id,
  c.namespace,
  c.id,
  c.name,
  c.description,
  c.metadata,
  c.tax_behavior,
  c.tax_code_id,
  now(),
  c.service_period_from,
  c.service_period_to,
  c.full_service_period_from,
  c.full_service_period_to,
  c.billing_period_from,
  c.billing_period_to,
  c.invoice_at,
  c.feature_key,
  c.price,
  c.discounts,
  c.unit_config
FROM (
  SELECT DISTINCT namespace, charge_id
  FROM repair_usagebased_deleted_invoice_runs
  WHERE run_type = 'final_realization'
) affected
JOIN charge_usage_based c
  ON c.namespace = affected.namespace
 AND c.id = affected.charge_id
ON CONFLICT (charge_id) DO UPDATE
SET intent_deleted_at = now();

UPDATE charge_usage_based c
SET
  deleted_at = now(),
  updated_at = now(),
  status = 'deleted',
  status_detailed = 'deleted'
FROM (
  SELECT DISTINCT namespace, charge_id
  FROM repair_usagebased_deleted_invoice_runs
  WHERE run_type = 'final_realization'
) affected
WHERE c.namespace = affected.namespace
  AND c.id = affected.charge_id
  AND c.deleted_at IS NULL;

COMMIT;
