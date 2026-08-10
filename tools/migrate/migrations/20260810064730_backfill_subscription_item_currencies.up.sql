-- Materialize the inherited subscription fiat currency on legacy priced items.
-- Mark modified rows so the data change is attributable and reversible.
BEGIN;

UPDATE subscription_items AS item
SET
  currency = subscription.currency,
  annotations =
    CASE
      WHEN item.annotations IS NULL OR item.annotations = 'null'::jsonb
        THEN '{}'::jsonb
      ELSE item.annotations
    END ||
    jsonb_build_object(
      'dbmigration:backfill_subscription_item_currencies',
      to_char(
        CURRENT_TIMESTAMP AT TIME ZONE 'UTC',
        'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'
      )
    ),
  updated_at = CURRENT_TIMESTAMP
FROM subscription_phases AS phase
JOIN subscriptions AS subscription
  ON subscription.id = phase.subscription_id
 AND subscription.namespace = phase.namespace
WHERE item.phase_id = phase.id
  AND item.namespace = phase.namespace
  AND item.price IS NOT NULL
  AND item.currency IS NULL
  AND item.custom_currency_id IS NULL
  AND NOT COALESCE(
    item.annotations ? 'dbmigration:backfill_subscription_item_currencies',
    false
  );

COMMIT;
