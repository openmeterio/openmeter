-- We'll update all SubscriptionItems that have a boolean entitlement associated with them
-- by merging their current annotations with the boolean entitlement count annotation: "subscription.entitlement.boolean.count"
-- if it's already present, we'll skip the item
-- if it's not present, we'll set its value to 1

UPDATE subscription_items si
SET annotations = CASE
    -- If there are no annotations yet, create a new object with the boolean count
    WHEN si.annotations IS NULL THEN '{"subscription.entitlement.boolean.count": 1}'::jsonb
    -- If annotations exist but don't have the boolean count key, add it
    WHEN NOT (si.annotations ? 'subscription.entitlement.boolean.count') THEN
        jsonb_set(si.annotations, '{subscription.entitlement.boolean.count}', '1'::jsonb)
    -- Otherwise keep the existing annotations (skip items that already have the count)
    ELSE si.annotations
END
FROM entitlements e
WHERE si.entitlement_id = e.id
AND e.entitlement_type = 'boolean';

