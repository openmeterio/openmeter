BEGIN;

UPDATE subscription_items
SET
  currency = NULL,
  annotations = CASE
    WHEN annotations - 'dbmigration:backfill_subscription_item_currencies' = '{}'::jsonb
      THEN NULL
    ELSE annotations - 'dbmigration:backfill_subscription_item_currencies'
  END,
  updated_at = CURRENT_TIMESTAMP
WHERE annotations ? 'dbmigration:backfill_subscription_item_currencies';

COMMIT;
