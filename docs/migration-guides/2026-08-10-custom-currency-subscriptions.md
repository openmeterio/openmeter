# Custom-currency subscriptions

This release is a breaking database upgrade for installations with existing subscriptions. Deploy the preceding expand-and-write release first: it materializes currency on every newly created or rewritten priced subscription item while continuing to read legacy rows.

Next, deploy the release containing migration `20260810064730_backfill_subscription_item_currencies`. It materializes the subscription's existing fiat currency on legacy priced items, which predate rate-card currency overrides, and marks the changed rows for an attributable rollback.

Before deploying the semantic release and applying migration `20260810084018_custom_currency_subscription_semantics`, verify that no legacy priced items remain without a currency:

```sql
-- This count must reach zero before the strict semantic migration is applied.
SELECT count(*) AS priced_items_without_currency
FROM subscription_items
WHERE price IS NOT NULL
  AND currency IS NULL
  AND custom_currency_id IS NULL;
```

If the query still returns rows, investigate why the automatic backfill could not resolve them. Legacy rows with a valid subscription relationship can be repaired with the following recovery query:

```sql
UPDATE subscription_items AS item
SET currency = subscription.currency,
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
  AND item.custom_currency_id IS NULL;
```

Run the verification query again after recovery. The strict semantic migration deliberately performs no backfill: it fails while a priced item has neither a fiat nor a custom-currency reference. Unpriced items must keep both currency columns null.
