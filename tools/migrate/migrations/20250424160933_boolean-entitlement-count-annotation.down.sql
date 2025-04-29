-- Remove the "subscription.entitlement.boolean.count" annotation from subscription items
-- that have a boolean entitlement associated with them

-- First, set annotations to NULL for items that only have the boolean count annotation
UPDATE subscription_items si
SET annotations = NULL
FROM entitlements e
WHERE si.entitlement_id = e.id
AND e.entitlement_type = 'boolean'
AND si.annotations::text = '{"subscription.entitlement.boolean.count": 1}';

-- Then, remove the boolean count annotation from items that have other annotations too
UPDATE subscription_items si
SET annotations = si.annotations - 'subscription.entitlement.boolean.count'
FROM entitlements e
WHERE si.entitlement_id = e.id
AND e.entitlement_type = 'boolean'
AND si.annotations ? 'subscription.entitlement.boolean.count'
AND si.annotations::text != '{"subscription.entitlement.boolean.count": 1}';
