# Deprecating Unaligned Subscriptions

From Version: `v1.0.0-beta.214`
To Version: `v1.0.0-beta.215`

## Summary

Unaligned subscriptions are being deprecated. As the first step, OpenMeter will [error](/openmeter/productcatalog/errors.go#L411) when creating a new unaligned subscription. At a later point (`v1.0.0-beta.216`), alignment config (`BillablesMustAlign`) will be removed and all subscriptions will follow the current aligned behavior.

This document describes the manual migration steps if you are using unaligned subscriptions. For more information on the alignment behavior, see the [documentation](https://openmeter.io/docs/billing/subscription/overview).

> We recommend you first read through this document before copy-pasting any queries. Be nice to your database.
## Migration Steps

The aim of the process is to convert all unaligned subscriptions (and plans) to aligned subscriptions (and plans) while minimizing the guesswork required.

While the generated cadences will be different between aligned and unaligned plans, an unaligned plan can be converted to an aligned plan if its cadence (BillingCadence) and every billable RateCard's billing cadence (ServiceCadence) meet the following criteria:

```md
For each ServiceCadence, either:
- the BillingCadence is a multiple of the ServiceCadence
- the ServiceCadence is a multiple of the BillingCadence
- the two cadences are the same
```

That check is otherwise implemented [here](/openmeter/productcatalog/alignment.go#L22).

One could write a PLPGSQL function to exhaustively check this criteria, but in our experience most unaligned plans (and subscriptions) have RateCards that share the same literal value with the Plan's BillingCadence. For simplicity, we'll only go with these cases.

### Plans

First, we'll collect all plans that have `BillablesMustAlign` set to `false`, and check whether they can be converted to aligned subscriptions.

```sql
-- Let's see the number of unaligned plans
SELECT COUNT(*) FROM plans WHERE billables_must_align = false;

-- Let's see their contents by namespace (openmeter uses default namespace for all resources)
SELECT
    p.namespace,
    jsonb_agg(p.*) AS plans
FROM
    plans p
WHERE
    p.billables_must_align = false
GROUP BY
    p.namespace
ORDER BY
    namespace ASC;
```

Now let's see the ones that can be converted to aligned plans.

```sql
-- Find plans where EVERY rate card has the same billing cadence as the plan
SELECT count(*)
FROM plans p
WHERE p.billables_must_align = false
AND NOT EXISTS (
    SELECT 1
    FROM plan_phases ph
    JOIN plan_rate_cards rc ON rc.phase_id = ph.id
    WHERE ph.plan_id = p.id
    AND p.billing_cadence != rc.billing_cadence
);

-- These can be simply updated with
UPDATE plans
SET billables_must_align = true
WHERE id IN (
  SELECT id
  FROM plans p
  WHERE p.billables_must_align = false
  AND NOT EXISTS (
      SELECT 1
      FROM plan_phases ph
      JOIN plan_rate_cards rc ON rc.phase_id = ph.id
      WHERE ph.plan_id = p.id
      AND p.billing_cadence != rc.billing_cadence
  )
);
```

Some plans will still require manual resolution. These will be the ones not returned by the above query
```sql
-- These will require manual resolution
SELECT namespace,id, key
FROM plans p
WHERE p.billables_must_align = false
AND EXISTS (
    SELECT 1
    FROM plan_phases ph
    JOIN plan_rate_cards rc ON rc.phase_id = ph.id
    WHERE ph.plan_id = p.id
    AND p.billing_cadence != rc.billing_cadence
);
```

### Subscriptions

Second, we'll do a similar process for subscriptions.

```sql
-- Let's see the number of unaligned subscriptions
SELECT COUNT(*) FROM subscriptions WHERE billables_must_align = false;

-- Let's see their contents by namespace (openmeter uses default namespace for all resources)
SELECT
    p.namespace,
    jsonb_agg(s.*) AS subscriptions
FROM
    subscriptions s
WHERE
    s.billables_must_align = false
GROUP BY
    s.namespace
ORDER BY
    namespace ASC;
```

And let's see which can be converted to aligned subscriptions.
```sql
SELECT count(*)
FROM subscriptions s
WHERE s.billables_must_align = false
AND NOT EXISTS (
    SELECT 1
    FROM subscription_phases ph
    JOIN subscription_items si ON si.phase_id = ph.id
    WHERE ph.subscription_id = s.id
    AND s.billing_cadence != si.billing_cadence
);

-- These can be simply updated with
UPDATE subscriptions
SET billables_must_align = true
WHERE id IN (
  SELECT id
  FROM subscriptions s
  WHERE s.billables_must_align = false
  AND NOT EXISTS (
    SELECT 1
    FROM subscription_phases ph
    JOIN subscription_items si ON si.phase_id = ph.id
    WHERE ph.subscription_id = s.id
    AND s.billing_cadence != si.billing_cadence
  )
);
```

Some subscriptions will still require manual resolution. These will be the ones not returned by the above query
```sql
-- These will require manual resolution
SELECT namespace, id, name
FROM subscriptions s
WHERE s.billables_must_align = false
AND EXISTS (
    SELECT 1
    FROM subscription_phases ph
    JOIN subscription_items si ON si.phase_id = ph.id
    WHERE ph.subscription_id = s.id
    AND s.billing_cadence != si.billing_cadence
);
```

After all these are addressed, we're done!