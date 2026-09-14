-- Remove subscription-managed usage charges whose standard invoice line was
-- persisted without the realization run required to own that line. Limiting
-- the repair to draft.invalid_created invoices keeps immutable invoice history
-- out of scope; requiring no run history avoids rewriting any realization.
BEGIN;

CREATE TEMPORARY TABLE repair_invalid_created_usage_charges ON COMMIT DROP AS
SELECT
  line.namespace,
  line.id AS line_id,
  usage_charge.id AS charge_id,
  usage_charge.subscription_id
FROM billing_invoice_lines AS line
JOIN billing_invoices AS invoice
  ON invoice.namespace = line.namespace
 AND invoice.id = line.invoice_id
JOIN charge_usage_based AS usage_charge
  ON usage_charge.namespace = line.namespace
 AND usage_charge.id = line.charge_id
JOIN charges AS charge
  ON charge.namespace = usage_charge.namespace
 AND charge.id = usage_charge.id
 AND charge.charge_usage_based_id = usage_charge.id
WHERE invoice.deleted_at IS NULL
  AND invoice.type = 'standard'
  AND invoice.status = 'draft.invalid_created'
  AND line.deleted_at IS NULL
  AND line.parent_line_id IS NULL
  AND line.status = 'valid'
  AND line.engine = 'charge_usagebased'
  AND line.managed_by = 'subscription'
  AND usage_charge.deleted_at IS NULL
  AND usage_charge.intent_deleted_at IS NULL
  AND usage_charge.settlement_mode = 'credit_then_invoice'
  AND usage_charge.managed_by = 'subscription'
  AND usage_charge.subscription_id IS NOT NULL
  AND usage_charge.unique_reference_id IS NOT NULL
  AND charge.deleted_at IS NULL
  AND charge.type = 'usage_based'
  AND NOT EXISTS (
    SELECT 1
    FROM charge_usage_based_runs AS run
    WHERE run.namespace = usage_charge.namespace
      AND run.charge_id = usage_charge.id
  );

UPDATE billing_invoice_lines AS line
SET
  deleted_at = now(),
  updated_at = now()
FROM repair_invalid_created_usage_charges AS affected
WHERE line.namespace = affected.namespace
  AND line.id = affected.line_id;

UPDATE charge_usage_based_overrides AS intent_override
SET intent_deleted_at = now()
FROM repair_invalid_created_usage_charges AS affected
WHERE intent_override.namespace = affected.namespace
  AND intent_override.charge_id = affected.charge_id;

UPDATE charge_usage_based AS usage_charge
SET
  intent_deleted_at = now(),
  deleted_at = now(),
  updated_at = now(),
  status = 'deleted',
  status_detailed = 'deleted'
FROM repair_invalid_created_usage_charges AS affected
WHERE usage_charge.namespace = affected.namespace
  AND usage_charge.id = affected.charge_id;

UPDATE charges AS charge
SET deleted_at = now()
FROM repair_invalid_created_usage_charges AS affected
WHERE charge.namespace = affected.namespace
  AND charge.id = affected.charge_id;

DELETE FROM subscription_billing_sync_states AS sync_state
USING repair_invalid_created_usage_charges AS affected
WHERE sync_state.namespace = affected.namespace
  AND sync_state.subscription_id = affected.subscription_id;

COMMIT;
