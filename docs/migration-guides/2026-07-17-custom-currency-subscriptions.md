# Custom-currency subscriptions

This release is a breaking database upgrade for installations with existing subscriptions. Deploy the preceding expand-and-write release first: it materializes currency on every newly created or rewritten priced subscription item while continuing to read legacy rows.

Before applying migration `20260717195001`, backfill the remaining legacy priced items. Those items predate rate-card currency overrides, so their currency is the subscription's existing fiat currency.

```sql
-- This count must reach zero before the migration is applied.
SELECT count(*) AS priced_items_without_currency
FROM subscription_items
WHERE price IS NOT NULL
  AND currency IS NULL
  AND custom_currency_id IS NULL;

UPDATE subscription_items AS item
SET currency = subscription.currency,
    updated_at = now()
FROM subscription_phases AS phase
JOIN subscriptions AS subscription
  ON subscription.id = phase.subscription_id
 AND subscription.namespace = phase.namespace
WHERE item.phase_id = phase.id
  AND item.namespace = phase.namespace
  AND item.price IS NOT NULL
  AND item.currency IS NULL
  AND item.custom_currency_id IS NULL;

SELECT count(*) AS priced_items_without_currency
FROM subscription_items
WHERE price IS NOT NULL
  AND currency IS NULL
  AND custom_currency_id IS NULL;
```

The migration deliberately performs no automatic backfill. It fails while a priced item has neither a fiat nor a custom-currency reference. Unpriced items must keep both currency columns null.
